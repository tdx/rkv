package bitcask

import (
	"io"

	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Backuper = (*svc)(nil)

//
// dbApi.Backuper interface implementation
//
func (s *svc) Backup(w io.Writer) error {
	// TODO:
	return nil
}
