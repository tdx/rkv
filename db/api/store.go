package api

// Entry describes entity in db
type Entry struct {
	Tab []byte
	Key []byte
	Val []byte
}

// BatchEntry for batch update
type BatchEntry struct {
	Operation Operation
	Entry     *Entry
	Err       error
}

// Operation type
type Operation int

// Constants for operation type
const (
	PutOperation    Operation = 0
	DeleteOperation           = 1
	GetOperation              = 2
	// TODO: ListOperation
)

// Storer interface to operate with data
type Storer interface {
	Put(tab, key, value []byte) error
	Get(tab, key []byte) ([]byte, error)
	Delete(tab, key []byte) error
	Batch([]*BatchEntry) error
}
