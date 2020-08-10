package bolt

import (
	"io"
	"time"

	dbApi "github.com/tdx/rkv/db/api"

	"github.com/boltdb/bolt"
	"github.com/rboyer/safeio"
)

var _ dbApi.Restorer = (*svc)(nil)

//
// dbApi.Restorer interface implementation
//

// Restore bolt file from reader
func (s *svc) Restore(r io.ReadCloser) error {

	newDb, err := installSnapshot(s.dir, r)
	if err != nil {
		return err
	}

	oldDb := s.db

	s.mu.Lock()
	s.db = newDb
	s.mu.Unlock()

	_ = oldDb.Close()

	return nil
}

func installSnapshot(dir string, r io.ReadCloser) (*bolt.DB, error) {

	name := fileName(dir)
	if _, err := safeio.WriteToFile(r, name, 0600); err != nil {
		return nil, err
	}

	return bolt.Open(name, 0600, &bolt.Options{Timeout: 1 * time.Second})
}
