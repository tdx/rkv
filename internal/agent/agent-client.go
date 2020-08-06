package agent

import "rkv/api"

var _ api.Client = (*Agent)(nil)

// Put ...
func (a *Agent) Put(tab, key, value []byte) error {
	return a.db.Put(tab, key, value)
}

// Get ...
func (a *Agent) Get(tab, key []byte) ([]byte, error) {
	return a.db.Get(tab, key)
}

// Delete ...
func (a *Agent) Delete(tab, key []byte) error {
	return a.db.Delete(tab, key)
}
