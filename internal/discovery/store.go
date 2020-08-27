package discovery

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type store struct {
	Joins string `json:"join-addrs"`
}

// Config stores/reads runtime data to/from file

type persist struct {
	configFile string
	mu         sync.RWMutex
	s          *store
}

func newPersist(dir string) (*persist, error) {

	p := &persist{
		configFile: filepath.Join(dir, "rkv-rt.json"),
		s:          &store{},
	}

	b, err := ioutil.ReadFile(p.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return nil, err
	}

	var s store
	if err = json.Unmarshal(b, &s); err != nil {
		return nil, err
	}

	p.s = &s

	return p, nil
}

// SaveJoins saves key with val in config file
func (p *persist) SaveJoins(addrs string) error {

	s := store{
		Joins: addrs,
	}
	b, err := json.Marshal(&s)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(p.configFile, b, 0644)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.s.Joins = addrs

	return nil
}

// Get reads config value by key
func (p *persist) GetJoins() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.s.Joins
}
