package bitcask

import (
	"io"

	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Restorer = (*svc)(nil)

//
// dbApi.Restorer interface implementation
//

func (s *svc) Restore(r io.ReadCloser) error {

	return nil
}
