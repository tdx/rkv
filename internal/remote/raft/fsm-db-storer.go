package raft

import (
	dbApi "github.com/tdx/rkv/internal/db/api"
)

var _ dbApi.Storer = (*fsm)(nil)
var _ dbApi.Closer = (*fsm)(nil)

// Put ...
func (f *fsm) Put(tab, key, val []byte) error {
	return f.db.Put(tab, key, val)
}

// Get ...
func (f *fsm) Get(tab, key []byte) ([]byte, error) {
	return f.db.Get(tab, key)
}

// Delete ...
func (f *fsm) Delete(tab, key []byte) error {
	return f.db.Delete(tab, key)
}

// Close ...
func (f *fsm) Close() error {
	return f.db.Close()
}
