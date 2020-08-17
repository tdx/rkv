package api

import (
	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
)

// Backend interface for disctributed database
type Backend interface {
	Put(tab, key, value []byte) error
	Get(level rkvApi.ConsistencyLevel, tab, key []byte) ([]byte, error)
	Delete(tab, key []byte) error
	Batch([]*dbApi.BatchEntry) error
	dbApi.Closer
}
