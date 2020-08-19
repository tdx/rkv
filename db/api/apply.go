package api

// ApplyFunc is a data handler func type
// See db/bolt|gmap/apply_test.go how to use
type ApplyFunc func(dbCtx interface{}, args []byte) (interface{}, error)

// Applier allows apply functions in raft cluster
// This interface implements DB which supports apply call
type Applier interface {
	Apply(fn ApplyFunc, args []byte, readOnly bool) (interface{}, error)
}
