package bolt

import (
	"github.com/boltdb/bolt"
	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Applier = (*svc)(nil)

func (s *svc) ApplyRead(
	fn dbApi.ApplyFunc, args ...[]byte) (interface{}, error) {

	var (
		err    error
		result interface{}
	)

	fb := func(tx *bolt.Tx) error {
		result, err = fn(tx, args...)
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	err = s.db.View(fb)
	return result, err
}

func (s *svc) ApplyWrite(
	fn dbApi.ApplyFunc, args ...[]byte) (interface{}, error) {

	var (
		err    error
		result interface{}
	)

	fb := func(tx *bolt.Tx) error {
		result, err = fn(tx, args...)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.db.Update(fb)

	return result, err
}
