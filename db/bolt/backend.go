package bolt

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	dbApi "github.com/tdx/rkv/db/api"

	"github.com/boltdb/bolt"
)

type svc struct {
	dir string
	db  *bolt.DB
	mu  sync.RWMutex
}

var _ dbApi.Backend = (*svc)(nil)

// New returns dbApi.Backend instance
func New(dir string) (dbApi.Backend, error) {

	db, err := open(dir)
	if err != nil {
		return nil, err
	}

	return &svc{
		dir: dir,
		db:  db,
	}, nil
}

func (s *svc) DSN() string {
	return s.db.Path()
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

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(tab)
		if err != nil {
			return err
		}

		return b.Put(key, value)
	})
}

func (s *svc) Get(tab, key []byte) ([]byte, error) {
	if tab == nil {
		return nil, dbApi.TableNilError
	}
	if key == nil {
		return nil, dbApi.KeyNilError
	}

	var valCopy []byte

	s.mu.RLock()
	defer s.mu.RUnlock()

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(tab)
		if b == nil {
			return dbApi.ErrNoTable(tab)
		}

		if val := b.Get(key); val != nil {
			valCopy = make([]byte, len(val))
			copy(valCopy, val)

			return nil
		}

		return dbApi.ErrNoKey(key)
	})

	if err != nil {
		return nil, err
	}

	return valCopy, nil
}

func (s *svc) Delete(tab, key []byte) error {
	if tab == nil {
		return dbApi.TableNilError
	}
	if key == nil {
		return dbApi.KeyNilError
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(tab)
		if b == nil {
			return dbApi.ErrNoTable(tab)
		}

		return b.Delete(key)
	})
}

func (s *svc) Batch(commands [][]*dbApi.BatchEntry, ro bool) error {

	fn := func(tx *bolt.Tx) error {
		for i := range commands {
			cmds := commands[i]
			for j := range cmds {
				cmd := cmds[j]
				switch cmd.Operation {
				case dbApi.PutOperation:
					b, err := tx.CreateBucketIfNotExists(cmd.Entry.Tab)
					if err != nil {
						return err
					}
					if err := b.Put(cmd.Entry.Key, cmd.Entry.Val); err != nil {
						return err
					}

				case dbApi.DeleteOperation:
					b := tx.Bucket(cmd.Entry.Tab)
					if b == nil {
						cmd.Result = dbApi.ErrNoTable(cmd.Entry.Tab)
						continue
					}
					if err := b.Delete(cmd.Entry.Key); err != nil {
						return err
					}

				case dbApi.GetOperation:
					b := tx.Bucket(cmd.Entry.Tab)
					if b == nil {
						cmd.Result = dbApi.ErrNoTable(cmd.Entry.Tab)
						continue
					}

					val := b.Get(cmd.Entry.Key)
					if val == nil {
						cmd.Result = dbApi.ErrNoKey(cmd.Entry.Key)
						continue
					}

					valCopy := make([]byte, len(val))
					copy(valCopy, val)
					cmd.Result = valCopy

				case dbApi.ApplyOperation:
					if cmd.Apply.Fn != nil {
						var err error
						cmd.Result, err = cmd.Apply.Fn(tx, cmd.Apply.Args...)
						if err != nil {
							cmd.Result = err
						}
					}
				}
			}
		}

		return nil
	}

	if ro {
		s.mu.RLock()
		defer s.mu.RUnlock()

		return s.db.View(fn)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(fn)
}

//
// dbApi.Closer umplementation
//
func (s *svc) Close() error {
	return s.db.Close()
}

//
func open(dir string) (*bolt.DB, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// first run
			if err := os.MkdirAll(dir, 0700); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	if len(files) == 0 {
		name := fileName(dir)
		return bolt.Open(name, 0600, &bolt.Options{Timeout: 1 * time.Second})
	}

	// find latest db file in 'dir' and remove old ones
	var lastDbFile string
	lastTime := time.Time{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !strings.HasPrefix(file.Name(), "bolt.db") {
			continue
		}

		fileTime := file.ModTime()
		if fileTime.After(lastTime) {
			lastTime = fileTime
			lastDbFile = file.Name()
		}
	}

	if lastDbFile == "" {
		name := fileName(dir)
		return bolt.Open(name, 0600, &bolt.Options{Timeout: 1 * time.Second})
	}

	// remove old files
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if file.Name() != lastDbFile {
			name := filepath.Join(dir, file.Name())
			_ = os.Remove(name)
		}
	}

	name := filepath.Join(dir, lastDbFile)

	return bolt.Open(name, 0600, &bolt.Options{Timeout: 1 * time.Second})
}

func fileName(dir string) string {
	const dbFile = "bolt.db"
	return filepath.Join(
		dir,
		dbFile+"-"+time.Now().Format("2006-01-02T15:04:05.000000"))
}
