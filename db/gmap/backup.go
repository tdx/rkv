package gmap

import (
	"encoding/gob"
	"io"

	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Backuper = (*svc)(nil)

//
// dbApi.Backuper interface implementation
//
func (s *svc) Backup(w io.Writer) error {

	e := gob.NewEncoder(w)

	s.mu.RLock()
	defer s.mu.RUnlock()

	return e.Encode(s.tabs)
}
