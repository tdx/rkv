package bitcask

import (
	dbApi "github.com/tdx/rkv/db/api"

	"github.com/prologic/bitcask"
)

type svc struct {
	dir string
	db  *bitcask.Bitcask
}

var _ dbApi.Backend = (*svc)(nil)

// New returns dbApi.Backend instance
func New(
	dir string,
	maxDatafileSize int) (dbApi.Backend, error) {

	db, err := bitcask.Open(
		dir,
		bitcask.WithMaxDatafileSize(maxDatafileSize),
	)
	if err != nil {
		return nil, err
	}

	return &svc{
		db:  db,
		dir: dir,
	}, nil
}

//
// dbApi.Storer umplementation
//

func (s *svc) DSN() string {
	return s.db.Path()
}

// tab ignored for bitcask
func (s *svc) Put(tab, key, value []byte) error {
	return s.db.Put(key, value)
}

// tab ignored for bitcask
func (s *svc) Get(tab, key []byte) ([]byte, error) {
	r, err := s.db.Get(key)
	if err == bitcask.ErrKeyNotFound {
		return nil, dbApi.ErrNoKey(key)
	}
	return r, err
}

// tab ignored for bitcask
func (s *svc) Delete(tab, key []byte) error {
	return s.db.Delete(key)
}

//
// dbApi.Closer umplementation
//
func (s *svc) Close() error {
	return s.db.Close()
}
