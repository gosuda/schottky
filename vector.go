package schottky

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

const (
	// ProjectionProfileVersion identifies the serialized projection-profile format.
	ProjectionProfileVersion = 1
	projectionProfileHeader  = 16
	projectionProfileMagic   = "SKVP"
)

// ProjectionMethod identifies how a projection profile was produced.
type ProjectionMethod uint8

const (
	ProjectionGaussian ProjectionMethod = iota + 1
	ProjectionPCA
)

// VectorNormalization identifies preprocessing applied before projection.
type VectorNormalization uint8

const (
	VectorUnnormalized VectorNormalization = iota
	VectorL2
)

// GaussianProjectionOptions configures a data-independent projection profile.
type GaussianProjectionOptions struct {
	Dimension     int
	Projections   int
	Seed          uint64
	Normalization VectorNormalization
}

// PCAOptions configures trained principal-component projections.
type PCAOptions struct {
	Projections   int
	Normalization VectorNormalization
	MaxIterations int
	Tolerance     float64
}

// ProjectionProfile is an immutable set of row-major Float32 projection axes.
// It is safe for concurrent use after construction.
type ProjectionProfile struct {
	method        ProjectionMethod
	normalization VectorNormalization
	dimension     int
	projections   int
	mean          []float32
	axes          []float32
}

// NewProjectionProfile constructs a profile from caller-supplied Float32 data.
// axes contains one contiguous dimension-length row per projection. PCA profiles
// require a dimension-length mean; Gaussian profiles require an empty mean.
func NewProjectionProfile(method ProjectionMethod, normalization VectorNormalization, dimension int, mean, axes []float32) (*ProjectionProfile, error) {
	return newProjectionProfile(method, normalization, dimension, mean, axes, true)
}

// NewGaussianProjectionProfile constructs independent Gaussian projection axes.
// The generated axes are initialization data, not a cross-language seed contract;
// persist the returned profile before writing projection keys.
func NewGaussianProjectionProfile(options GaussianProjectionOptions) (*ProjectionProfile, error) {
	if options.Dimension <= 0 || options.Projections <= 0 || !options.Normalization.valid() {
		return nil, ErrInvalidValue
	}
	if options.Dimension > maxInt()/options.Projections {
		return nil, ErrInvalidValue
	}

	axes := make([]float32, options.Dimension*options.Projections)
	random := splitMix64{state: options.Seed}
	for index := 0; index < len(axes); index += 2 {
		first := random.openUnit()
		second := random.openUnit()
		radius := math.Sqrt(-2 * math.Log(first))
		angle := 2 * math.Pi * second
		axes[index] = float32(radius * math.Cos(angle))
		if index+1 < len(axes) {
			axes[index+1] = float32(radius * math.Sin(angle))
		}
	}
	return newProjectionProfile(ProjectionGaussian, options.Normalization, options.Dimension, nil, axes, false)
}

// TrainPCA fits leading principal-component axes to samples. Training centers
// samples after applying the requested normalization and uses covariance power
// iteration without materializing a dimension-squared covariance matrix.
func TrainPCA(samples [][]float32, options PCAOptions) (*ProjectionProfile, error) {
	if len(samples) < 2 || options.Projections <= 0 || !options.Normalization.valid() {
		return nil, ErrInvalidValue
	}
	dimension := len(samples[0])
	if dimension == 0 || options.Projections > dimension {
		return nil, ErrInvalidValue
	}
	iterations := options.MaxIterations
	if iterations == 0 {
		iterations = 128
	}
	tolerance := options.Tolerance
	if tolerance == 0 {
		tolerance = 1e-8
	}
	if iterations < 1 || tolerance <= 0 || tolerance >= 1 {
		return nil, ErrInvalidValue
	}
	if len(samples) > maxInt()/dimension {
		return nil, ErrInvalidValue
	}

	inverseNorms := make([]float64, len(samples))
	mean64 := make([]float64, dimension)
	for rowIndex, sample := range samples {
		if len(sample) != dimension {
			return nil, ErrInvalidValue
		}
		normSquared := 0.0
		for _, component := range sample {
			value := float64(component)
			if !finite(value) {
				return nil, ErrInvalidValue
			}
			normSquared += value * value
		}
		if !finite(normSquared) {
			return nil, ErrInvalidValue
		}
		scale := 1.0
		if options.Normalization == VectorL2 {
			if normSquared == 0 {
				return nil, ErrInvalidValue
			}
			scale = 1 / math.Sqrt(normSquared)
		}
		inverseNorms[rowIndex] = scale
		for componentIndex, component := range sample {
			mean64[componentIndex] += float64(component) * scale
		}
	}
	inverseSampleCount := 1 / float64(len(samples))
	mean := make([]float32, dimension)
	for index := range mean64 {
		mean[index] = float32(mean64[index] * inverseSampleCount)
		mean64[index] = float64(mean[index])
	}

	centered := make([]float32, len(samples)*dimension)
	for rowIndex, sample := range samples {
		row := centered[rowIndex*dimension : (rowIndex+1)*dimension]
		scale := inverseNorms[rowIndex]
		for componentIndex, component := range sample {
			value := float32(float64(component)*scale - mean64[componentIndex])
			if !finite(float64(value)) {
				return nil, ErrInvalidValue
			}
			row[componentIndex] = value
		}
	}

	trainedAxes := make([]float64, options.Projections*dimension)
	current := make([]float64, dimension)
	next := make([]float64, dimension)
	random := splitMix64{state: uint64(dimension)<<32 ^ uint64(len(samples)) ^ uint64(options.Projections)}
	for projection := 0; projection < options.Projections; projection++ {
		for index := range current {
			current[index] = random.signedUnit()
		}
		if !orthonormalize(current, trainedAxes, projection, dimension) {
			return nil, ErrInvalidValue
		}

		converged := false
		for range iterations {
			clear(next)
			for rowStart := 0; rowStart < len(centered); rowStart += dimension {
				row := centered[rowStart : rowStart+dimension]
				dot := 0.0
				for index, component := range row {
					dot += float64(component) * current[index]
				}
				for index, component := range row {
					next[index] += float64(component) * dot
				}
			}
			if !orthonormalize(next, trainedAxes, projection, dimension) {
				return nil, ErrInvalidValue
			}
			alignment := 0.0
			for index := range current {
				alignment += current[index] * next[index]
			}
			copy(current, next)
			if 1-math.Abs(alignment) <= tolerance {
				converged = true
				break
			}
		}
		if !converged {
			return nil, ErrPCAConvergence
		}
		canonicalizeAxisSign(current)
		copy(trainedAxes[projection*dimension:(projection+1)*dimension], current)
	}

	axes := make([]float32, len(trainedAxes))
	for index, value := range trainedAxes {
		axes[index] = float32(value)
	}
	return newProjectionProfile(ProjectionPCA, options.Normalization, dimension, mean, axes, false)
}

// Method returns the profile construction method.
func (p *ProjectionProfile) Method() ProjectionMethod {
	if p == nil {
		return 0
	}
	return p.method
}

// Normalization returns the preprocessing required before projection.
func (p *ProjectionProfile) Normalization() VectorNormalization {
	if p == nil {
		return VectorUnnormalized
	}
	return p.normalization
}

// Dimension returns the required vector dimension.
func (p *ProjectionProfile) Dimension() int {
	if p == nil {
		return 0
	}
	return p.dimension
}

// ProjectionCount returns the number of scalar projection keys per vector.
func (p *ProjectionProfile) ProjectionCount() int {
	if p == nil {
		return 0
	}
	return p.projections
}

// Project appends all scalar projections to dst without growing it. On error it
// returns dst unchanged.
func (p *ProjectionProfile) Project(dst []float32, vector []float32) ([]float32, error) {
	if p == nil || len(vector) != p.dimension {
		return dst, ErrInvalidValue
	}
	normSquared := 0.0
	for _, component := range vector {
		value := float64(component)
		if !finite(value) {
			return dst, ErrInvalidValue
		}
		normSquared += value * value
	}
	if !finite(normSquared) {
		return dst, ErrInvalidValue
	}
	scale := 1.0
	if p.normalization == VectorL2 {
		if normSquared == 0 {
			return dst, ErrInvalidValue
		}
		scale = 1 / math.Sqrt(normSquared)
	}
	if p.projections > cap(dst)-len(dst) {
		return dst, ErrShortBuffer
	}

	start := len(dst)
	dst = dst[:start+p.projections]
	for projection := 0; projection < p.projections; projection++ {
		axis := p.axes[projection*p.dimension : (projection+1)*p.dimension]
		dot := 0.0
		for index, component := range vector {
			value := float64(component) * scale
			if len(p.mean) != 0 {
				value -= float64(p.mean[index])
			}
			dot += value * float64(axis[index])
		}
		projected := float32(dot)
		if !finite(dot) || !finite(float64(projected)) {
			return dst[:start], ErrInvalidValue
		}
		dst[start+projection] = projected
	}
	return dst, nil
}

// MarshalBinary returns the canonical serialized profile.
func (p *ProjectionProfile) MarshalBinary() ([]byte, error) {
	if p == nil {
		return nil, ErrInvalidValue
	}
	floatCount := len(p.mean) + len(p.axes)
	if floatCount > (maxInt()-projectionProfileHeader)/4 {
		return nil, ErrInvalidValue
	}
	encoded := make([]byte, projectionProfileHeader+4*floatCount)
	copy(encoded[:4], projectionProfileMagic)
	encoded[4] = ProjectionProfileVersion
	encoded[5] = byte(p.method)
	encoded[6] = byte(p.normalization)
	if len(p.mean) != 0 {
		encoded[7] = 1
	}
	binary.BigEndian.PutUint32(encoded[8:12], uint32(p.dimension))
	binary.BigEndian.PutUint32(encoded[12:16], uint32(p.projections))
	offset := projectionProfileHeader
	for _, values := range [][]float32{p.mean, p.axes} {
		for _, value := range values {
			binary.BigEndian.PutUint32(encoded[offset:offset+4], math.Float32bits(value))
			offset += 4
		}
	}
	return encoded, nil
}

// ParseProjectionProfile validates and decodes a canonical serialized profile.
func ParseProjectionProfile(encoded []byte) (*ProjectionProfile, error) {
	if len(encoded) < projectionProfileHeader || string(encoded[:4]) != projectionProfileMagic || encoded[4] != ProjectionProfileVersion {
		return nil, ErrMalformedProfile
	}
	method := ProjectionMethod(encoded[5])
	normalization := VectorNormalization(encoded[6])
	flags := encoded[7]
	if flags&^byte(1) != 0 {
		return nil, ErrMalformedProfile
	}
	dimension64 := uint64(binary.BigEndian.Uint32(encoded[8:12]))
	projections64 := uint64(binary.BigEndian.Uint32(encoded[12:16]))
	if dimension64 == 0 || projections64 == 0 || dimension64 > uint64(maxInt()) || projections64 > uint64(maxInt()) || dimension64*projections64 > uint64(maxInt()) {
		return nil, ErrMalformedProfile
	}
	dimension := int(dimension64)
	meanCount := 0
	if flags&1 != 0 {
		meanCount = dimension
	}
	floatCount64 := dimension64*projections64 + uint64(meanCount)
	if floatCount64 > uint64((maxInt()-projectionProfileHeader)/4) || projectionProfileHeader+int(floatCount64)*4 != len(encoded) {
		return nil, ErrMalformedProfile
	}

	values := make([]float32, int(floatCount64))
	offset := projectionProfileHeader
	for index := range values {
		values[index] = math.Float32frombits(binary.BigEndian.Uint32(encoded[offset : offset+4]))
		offset += 4
	}
	mean := values[:meanCount:meanCount]
	axes := values[meanCount:]
	profile, err := newProjectionProfile(method, normalization, dimension, mean, axes, false)
	if err != nil {
		return nil, ErrMalformedProfile
	}
	return profile, nil
}

// Fingerprint returns the SHA-256 digest of the canonical serialized profile.
func (p *ProjectionProfile) Fingerprint() [32]byte {
	encoded, err := p.MarshalBinary()
	if err != nil {
		return [32]byte{}
	}
	return sha256.Sum256(encoded)
}

func newProjectionProfile(method ProjectionMethod, normalization VectorNormalization, dimension int, mean, axes []float32, copyValues bool) (*ProjectionProfile, error) {
	if !method.valid() || !normalization.valid() || dimension <= 0 || len(axes) == 0 || len(axes)%dimension != 0 {
		return nil, ErrInvalidValue
	}
	projections := len(axes) / dimension
	if uint64(dimension) > uint64(^uint32(0)) || uint64(projections) > uint64(^uint32(0)) {
		return nil, ErrInvalidValue
	}
	if method == ProjectionPCA {
		if len(mean) != dimension || projections > dimension {
			return nil, ErrInvalidValue
		}
	} else if len(mean) != 0 {
		return nil, ErrInvalidValue
	}
	for _, value := range mean {
		if !finite(float64(value)) {
			return nil, ErrInvalidValue
		}
	}
	for projection := 0; projection < projections; projection++ {
		normSquared := 0.0
		for _, value := range axes[projection*dimension : (projection+1)*dimension] {
			if !finite(float64(value)) {
				return nil, ErrInvalidValue
			}
			normSquared += float64(value) * float64(value)
		}
		if normSquared == 0 || !finite(normSquared) {
			return nil, ErrInvalidValue
		}
	}
	if copyValues {
		mean = append([]float32(nil), mean...)
		axes = append([]float32(nil), axes...)
	}
	return &ProjectionProfile{
		method:        method,
		normalization: normalization,
		dimension:     dimension,
		projections:   projections,
		mean:          mean,
		axes:          axes,
	}, nil
}

func orthonormalize(vector, axes []float64, completed, dimension int) bool {
	for projection := 0; projection < completed; projection++ {
		axis := axes[projection*dimension : (projection+1)*dimension]
		dot := 0.0
		for index := range vector {
			dot += vector[index] * axis[index]
		}
		for index := range vector {
			vector[index] -= dot * axis[index]
		}
	}
	normSquared := 0.0
	for _, value := range vector {
		normSquared += value * value
	}
	if normSquared == 0 || !finite(normSquared) {
		return false
	}
	inverseNorm := 1 / math.Sqrt(normSquared)
	for index := range vector {
		vector[index] *= inverseNorm
	}
	return true
}

func canonicalizeAxisSign(axis []float64) {
	largest := 0
	for index := 1; index < len(axis); index++ {
		if math.Abs(axis[index]) > math.Abs(axis[largest]) {
			largest = index
		}
	}
	if axis[largest] < 0 {
		for index := range axis {
			axis[index] = -axis[index]
		}
	}
}

func (method ProjectionMethod) valid() bool {
	return method == ProjectionGaussian || method == ProjectionPCA
}

func (normalization VectorNormalization) valid() bool {
	return normalization == VectorUnnormalized || normalization == VectorL2
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func maxInt() int {
	return int(^uint(0) >> 1)
}

type splitMix64 struct {
	state uint64
}

func (random *splitMix64) next() uint64 {
	random.state += 0x9e3779b97f4a7c15
	value := random.state
	value = (value ^ value>>30) * 0xbf58476d1ce4e5b9
	value = (value ^ value>>27) * 0x94d049bb133111eb
	return value ^ value>>31
}

func (random *splitMix64) openUnit() float64 {
	return (float64(random.next()>>11) + 0.5) * (1.0 / (1 << 53))
}

func (random *splitMix64) signedUnit() float64 {
	return 2*random.openUnit() - 1
}
