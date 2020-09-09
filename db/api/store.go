package api

// Entry describes entity in db
type Entry struct {
	Tab []byte
	Key []byte
	Val []byte
}

// Apply describe apply call
type Apply struct {
	Fn       ApplyFunc
	Args     [][]byte
	ReadOnly bool
}

// BatchEntry for batch update
type BatchEntry struct {
	Operation Operation
	Entry     *Entry
	Apply     *Apply
	Result    interface{}
}

// Operation type
type Operation int

// Constants for operation type
const (
	PutOperation    Operation = 0
	DeleteOperation           = 1
	GetOperation              = 2
	ApplyOperation            = 4
)

// Storer interface to operate with data
type Storer interface {
	Put(tab, key, value []byte) error
	Get(tab, key []byte) ([]byte, error)
	Delete(tab, key []byte) error
	Batch(commands [][]*BatchEntry, readOnly bool) error
}
