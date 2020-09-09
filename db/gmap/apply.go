package gmap

import (
	dbApi "github.com/tdx/rkv/db/api"
)

var _ dbApi.Applier = (*svc)(nil)

func (s *svc) ApplyRead(
	fn dbApi.ApplyFunc, args ...[]byte) (interface{}, error) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	return fn(s.tabs, args...)
}

func (s *svc) ApplyWrite(
	fn dbApi.ApplyFunc, args ...[]byte) (interface{}, error) {

	s.mu.Lock()
	defer s.mu.Unlock()

	return fn(s.tabs, args...)
}
