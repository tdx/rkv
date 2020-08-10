package api

// Backend is the interface required for a phisical backend.
type Backend interface {
	DSN() string
	Storer
	Closer
	Backuper
	Restorer
}
