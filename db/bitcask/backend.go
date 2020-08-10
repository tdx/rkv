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
func New(dir string) (dbApi.Backend, error) {
	db, err := bitcask.Open(dir)
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

func (s *svc) Put(tab, key, value []byte) error {
	return s.db.Put(key, value)
}

func (s *svc) Get(tab, key []byte) ([]byte, error) {
	return s.db.Get(key)
}

func (s *svc) Delete(tab, key []byte) error {
	return s.db.Delete(key)
}

//
// dbApi.Closer umplementation
//
func (s *svc) Close() error {
	return s.db.Close()
}
