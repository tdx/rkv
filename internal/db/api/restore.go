package api

import "io"

// Restorer interface to restore backend from reader
type Restorer interface {
	Restore(io.ReadCloser) error
}
