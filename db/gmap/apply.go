package gmap

import (
	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Applier = (*svc)(nil)

func (s *svc) Apply(
	fn dbApi.ApplyFunc, args []byte, readOnly bool) (interface{}, error) {

	if readOnly {
		s.mu.RLock()
		defer s.mu.RUnlock()
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	return fn(s.tabs, args)
}
