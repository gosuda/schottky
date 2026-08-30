package schottky

import "errors"

var (
	ErrInvalidOrder     = errors.New("schottky: invalid order")
	ErrInvalidValue     = errors.New("schottky: invalid value")
	ErrMalformedKey     = errors.New("schottky: malformed key")
	ErrMalformedProfile = errors.New("schottky: malformed profile")
	ErrPCAConvergence   = errors.New("schottky: PCA did not converge")
	ErrShortBuffer      = errors.New("schottky: short buffer")
)

// Presence reports whether a decoded field contains a value.
type Presence uint8

const (
	Present Presence = iota
	Null
)
