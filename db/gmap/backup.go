package gmap

import (
	"bytes"
	"encoding/gob"
	"io"

	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Backuper = (*svc)(nil)

//
// dbApi.Backuper interface implementation
//
func (s *svc) Backup(w io.Writer) error {

	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := e.Encode(s.tabs); err != nil {
		return err
	}

	_, err := w.Write(b.Bytes())

	return err
}
