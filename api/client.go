package api

// Client is a client interface to database
type Client interface {
	Put(tab, key, value []byte) error
	Get(tab, key []byte) ([]byte, error)
	Delete(tab, key []byte) error
	Shutdown() error
}
