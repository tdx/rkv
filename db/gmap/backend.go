package gmap

import (
	"reflect"
	"sync"
	"unsafe"

	dbApi "github.com/tdx/rkv/db/api"
)

type table map[string][]byte

type svc struct {
	dir  string
	tabs map[string]table
	mu   sync.RWMutex
}

var _ dbApi.Backend = (*svc)(nil)

// New returns dbApi.Backend instance
func New(dir string) (dbApi.Backend, error) {
	return &svc{
		dir:  dir,
		tabs: make(map[string]table),
	}, nil
}

func (s *svc) DSN() string {
	return s.dir
}

//
// dbApi.Storer umplementation
//
func (s *svc) Put(tab, key, value []byte) error {

	if tab == nil {
		return dbApi.TableNilError
	}
	if key == nil {
		return dbApi.KeyNilError
	}

	s.mu.RLock()
	t, ok := s.tabs[b2s(tab)]
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if !ok {
		t = make(table)
		s.tabs[b2s(tab)] = t
	}
	t[b2s(key)] = value

	return nil
}

func (s *svc) Get(tab, key []byte) ([]byte, error) {

	if tab == nil {
		return nil, dbApi.TableNilError
	}
	if key == nil {
		return nil, dbApi.KeyNilError
	}

	s.mu.RLock()
	t, ok := s.tabs[b2s(tab)]
	s.mu.RUnlock()

	if !ok {
		return nil, dbApi.ErrNoTable(tab)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := t[b2s(key)]
	if !ok {
		return nil, dbApi.ErrNoKey(key)
	}

	return v, nil
}

func (s *svc) Delete(tab, key []byte) error {
	if tab == nil {
		return dbApi.TableNilError
	}
	if key == nil {
		return dbApi.KeyNilError
	}

	s.mu.RLock()
	t, ok := s.tabs[b2s(tab)]
	s.mu.RUnlock()

	if !ok {
		return dbApi.ErrNoTable(tab)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(t, b2s(key))

	return nil
}

//
// dbApi.Closer umplementation
//
func (s *svc) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tabs = make(map[string]table)

	return nil
}

//
func b2s(b []byte) string {
	bytesHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	strHeader := reflect.StringHeader{Data: bytesHeader.Data, Len: bytesHeader.Len}
	return *(*string)(unsafe.Pointer(&strHeader))
}
