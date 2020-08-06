package api

// Storer interface to operate with data
type Storer interface {
	Put(tab, key, value []byte) error
	Get(tab, key []byte) ([]byte, error)
	Delete(tab, key []byte) error
}
