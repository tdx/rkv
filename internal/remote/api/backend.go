package api

import dbApi "github.com/tdx/rkv/internal/db/api"

// Backend interface for disctributed database
type Backend interface {
	dbApi.Storer
	dbApi.Closer
}
