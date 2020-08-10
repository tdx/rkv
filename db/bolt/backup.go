package bolt

import (
	"io"

	dbApi "github.com/tdx/rkv/db/api"

	"github.com/boltdb/bolt"
)

var _ dbApi.Backuper = (*svc)(nil)

//
// dbApi.Backuper interface implementation
//
func (s *svc) Backup(w io.Writer) error {
	return s.db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(w)
		return err
	})
}
