package gmap

import (
	"encoding/gob"
	"io"

	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Restorer = (*svc)(nil)

//
// dbApi.Restorer interface implementation
//

// Restore map from reader
func (s *svc) Restore(r io.ReadCloser) error {

	var tabs map[string]map[string][]byte

	d := gob.NewDecoder(r)

	if err := d.Decode(&tabs); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tabs = tabs

	return nil
}
