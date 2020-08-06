package api

import dbApi "rkv/internal/db/api"

// Backend interface for disctributed database
type Backend interface {
	dbApi.Storer
	dbApi.Closer
}
