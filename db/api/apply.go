package api

// ApplyFunc is a data handler func type
// See db/bolt|gmap/apply_test.go how to use
type ApplyFunc func(dbCtx interface{}, args ...[]byte) (interface{}, error)

// Applier allows apply functions in raft cluster
// This interface implements DB which supports apply call
type Applier interface {
	ApplyRead(fn ApplyFunc, args ...[]byte) (interface{}, error)
	ApplyWrite(fn ApplyFunc, args ...[]byte) (interface{}, error)
}
