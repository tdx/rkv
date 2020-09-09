package gmap

import (
	"path/filepath"
	"reflect"
	"sync"
	"unsafe"

	dbApi "github.com/tdx/rkv/db/api"
)

type svc struct {
	dir  string
	tabs map[string]map[string][]byte
	mu   sync.RWMutex
}

var _ dbApi.Backend = (*svc)(nil)

// New returns dbApi.Backend instance
func New(dir string) (dbApi.Backend, error) {
	return &svc{
		dir:  filepath.Join(dir, "inmem"),
		tabs: make(map[string]map[string][]byte),
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
		t = make(map[string][]byte)
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

func (s *svc) Batch(commands [][]*dbApi.BatchEntry, ro bool) error {

	if ro {
		s.mu.RLock()
		defer s.mu.RUnlock()
	} else {
		s.mu.Lock()
		defer s.mu.Unlock()
	}

	for i := range commands {
		cmds := commands[i]
		for j := range cmds {
			cmd := cmds[j]
			switch cmd.Operation {
			case dbApi.PutOperation:
				t, ok := s.tabs[b2s(cmd.Entry.Tab)]
				if !ok {
					t = make(map[string][]byte)
					s.tabs[b2s(cmd.Entry.Tab)] = t
				}
				t[b2s(cmd.Entry.Key)] = cmd.Entry.Val

			case dbApi.DeleteOperation:
				t, ok := s.tabs[b2s(cmd.Entry.Tab)]
				if !ok {
					cmd.Result = dbApi.ErrNoTable(cmd.Entry.Tab)
					continue
				}
				delete(t, b2s(cmd.Entry.Key))

			case dbApi.GetOperation:
				t, ok := s.tabs[b2s(cmd.Entry.Tab)]
				if !ok {
					cmd.Result = dbApi.ErrNoTable(cmd.Entry.Tab)
					continue
				}
				cmd.Result, ok = t[b2s(cmd.Entry.Key)]
				if !ok {
					cmd.Result = dbApi.ErrNoKey(cmd.Entry.Key)
				}

			case dbApi.ApplyOperation:
				if cmd.Apply.Fn != nil {
					var err error
					cmd.Result, err = cmd.Apply.Fn(s.tabs, cmd.Apply.Args...)
					if err != nil {
						cmd.Result = err
					}
				}
			}
		}
	}

	return nil
}

//
// dbApi.Closer umplementation
//
func (s *svc) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tabs = make(map[string]map[string][]byte)

	return nil
}

//
func b2s(b []byte) string {
	bytesHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	strHeader := reflect.StringHeader{Data: bytesHeader.Data, Len: bytesHeader.Len}
	return *(*string)(unsafe.Pointer(&strHeader))
}
