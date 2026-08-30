package schottky_test

import (
	"encoding/hex"
	"errors"
	"math"
	"slices"
	"testing"

	"gosuda.org/schottky"
)

func TestProjectionProfileProjectsWithoutGrowingDestination(t *testing.T) {
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorL2,
		2,
		nil,
		[]float32{1, 0, 0, 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	destination := make([]float32, 1, 3)
	destination[0] = 9
	projected, err := profile.Project(destination, []float32{3, 4})
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{9, 0.6, 0.8}
	if !slices.Equal(projected, want) {
		t.Fatalf("Project() = %v, want %v", projected, want)
	}

	short := make([]float32, 1, 2)
	unchanged, err := profile.Project(short, []float32{3, 4})
	if !errors.Is(err, schottky.ErrShortBuffer) {
		t.Fatalf("Project() error = %v, want ErrShortBuffer", err)
	}
	if len(unchanged) != len(short) {
		t.Fatalf("Project() length = %d, want %d", len(unchanged), len(short))
	}
}

func TestProjectionProfileCopiesConstructorData(t *testing.T) {
	axes := []float32{1, 0}
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorUnnormalized,
		2,
		nil,
		axes,
	)
	if err != nil {
		t.Fatal(err)
	}
	axes[0] = 0
	axes[1] = 1

	storage := make([]float32, 0, 1)
	projected, err := profile.Project(storage, []float32{3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if projected[0] != 3 {
		t.Fatalf("Project() = %v, want [3]", projected)
	}
}

func TestProjectionProfileSerializationRoundTrip(t *testing.T) {
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorL2,
		2,
		nil,
		[]float32{1, -2},
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := profile.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	const wantHex = "534b56500101010000000002000000013f800000c0000000"
	if hex.EncodeToString(encoded) != wantHex {
		t.Fatalf("MarshalBinary() = %x, want %s", encoded, wantHex)
	}

	parsed, err := schottky.ParseProjectionProfile(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Method() != schottky.ProjectionGaussian || parsed.Normalization() != schottky.VectorL2 || parsed.Dimension() != 2 || parsed.ProjectionCount() != 1 {
		t.Fatalf(
			"parsed metadata = (%d, %d, %d, %d)",
			parsed.Method(), parsed.Normalization(), parsed.Dimension(), parsed.ProjectionCount(),
		)
	}
	if parsed.Fingerprint() != profile.Fingerprint() {
		t.Fatal("parsed profile fingerprint differs from source")
	}
}

func TestPCAProfileSerializationPreservesMean(t *testing.T) {
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionPCA,
		schottky.VectorUnnormalized,
		2,
		[]float32{1, 2},
		[]float32{1, 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := profile.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := schottky.ParseProjectionProfile(encoded)
	if err != nil {
		t.Fatal(err)
	}
	projected, err := parsed.Project(make([]float32, 0, 1), []float32{3, 2})
	if err != nil {
		t.Fatal(err)
	}
	if projected[0] != 2 {
		t.Fatalf("Project() = %v, want [2]", projected)
	}
}

func TestProjectionProfileRejectsMalformedEncoding(t *testing.T) {
	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorUnnormalized,
		1,
		nil,
		[]float32{1},
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := profile.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	tests := [][]byte{
		encoded[:len(encoded)-1],
		append(slices.Clone(encoded), 0),
		slices.Clone(encoded),
	}
	copy(tests[2][16:20], []byte{0x7f, 0xc0, 0, 0})
	for _, malformed := range tests {
		if _, err := schottky.ParseProjectionProfile(malformed); !errors.Is(err, schottky.ErrMalformedProfile) {
			t.Fatalf("ParseProjectionProfile(%x) error = %v, want ErrMalformedProfile", malformed, err)
		}
	}
}

func TestGaussianProjectionProfileIsDeterministic(t *testing.T) {
	options := schottky.GaussianProjectionOptions{
		Dimension:     4,
		Projections:   3,
		Seed:          42,
		Normalization: schottky.VectorL2,
	}
	first, err := schottky.NewGaussianProjectionProfile(options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := schottky.NewGaussianProjectionProfile(options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Fatal("same Gaussian options produced different profiles")
	}

	options.Seed++
	third, err := schottky.NewGaussianProjectionProfile(options)
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() == third.Fingerprint() {
		t.Fatal("different Gaussian seeds produced the same profile")
	}
}

func TestTrainPCAFindsLeadingComponent(t *testing.T) {
	samples := [][]float32{
		{-4, -0.2},
		{-3, 0.1},
		{-2, -0.1},
		{-1, 0.2},
		{1, -0.2},
		{2, 0.1},
		{3, -0.1},
		{4, 0.2},
	}
	profile, err := schottky.TrainPCA(samples, schottky.PCAOptions{
		Projections:   1,
		Normalization: schottky.VectorUnnormalized,
		MaxIterations: 256,
		Tolerance:     1e-10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Method() != schottky.ProjectionPCA || profile.Dimension() != 2 || profile.ProjectionCount() != 1 {
		t.Fatalf("profile metadata = (%d, %d, %d)", profile.Method(), profile.Dimension(), profile.ProjectionCount())
	}

	positive, err := profile.Project(make([]float32, 0, 1), []float32{3, 0})
	if err != nil {
		t.Fatal(err)
	}
	negative, err := profile.Project(make([]float32, 0, 1), []float32{-3, 0})
	if err != nil {
		t.Fatal(err)
	}
	if positive[0] <= 2.9 || negative[0] >= -2.9 {
		t.Fatalf("PCA projections = (%g, %g), want dominant signed x-axis", positive[0], negative[0])
	}
}

func TestTrainPCAL2NormalizationMakesScaleIrrelevant(t *testing.T) {
	samples := [][]float32{{1, 0}, {2, 0}, {0, 1}, {0, 2}, {-1, 0}, {0, -1}}
	profile, err := schottky.TrainPCA(samples, schottky.PCAOptions{
		Projections:   1,
		Normalization: schottky.VectorL2,
		MaxIterations: 256,
		Tolerance:     1e-9,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := profile.Project(make([]float32, 0, 1), []float32{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	second, err := profile.Project(make([]float32, 0, 1), []float32{3, 6})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(first[0]-second[0])) > 1e-6 {
		t.Fatalf("scale-equivalent projections = (%g, %g)", first[0], second[0])
	}
}

func TestProjectionProfilesRejectInvalidInputs(t *testing.T) {
	if _, err := schottky.NewGaussianProjectionProfile(schottky.GaussianProjectionOptions{}); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("NewGaussianProjectionProfile() error = %v, want ErrInvalidValue", err)
	}
	if _, err := schottky.NewProjectionProfile(schottky.ProjectionPCA, schottky.VectorL2, 2, nil, []float32{1, 0}); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("NewProjectionProfile() error = %v, want ErrInvalidValue", err)
	}
	if _, err := schottky.TrainPCA([][]float32{{1, 2}, {1}}, schottky.PCAOptions{Projections: 1}); !errors.Is(err, schottky.ErrInvalidValue) {
		t.Fatalf("TrainPCA() error = %v, want ErrInvalidValue", err)
	}

	profile, err := schottky.NewProjectionProfile(
		schottky.ProjectionGaussian,
		schottky.VectorL2,
		2,
		nil,
		[]float32{1, 0},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, vector := range [][]float32{{0, 0}, {1}, {float32(math.NaN()), 0}} {
		if _, err := profile.Project(make([]float32, 0, 1), vector); !errors.Is(err, schottky.ErrInvalidValue) {
			t.Fatalf("Project(%v) error = %v, want ErrInvalidValue", vector, err)
		}
	}
}
