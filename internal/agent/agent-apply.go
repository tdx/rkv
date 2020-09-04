package agent

import (
	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
)

var _ rkvApi.ApplyRegistrator = (*Agent)(nil)

// RegisterApplyRead ...
func (a *Agent) RegisterApplyRead(name string, fn dbApi.ApplyFunc) error {
	return a.registry.RegisterApplyRead(name, fn)
}

// RegisterApplyWrite ...
func (a *Agent) RegisterApplyWrite(name string, fn dbApi.ApplyFunc) error {
	return a.registry.RegisterApplyWrite(name, fn)
}

// UnRegisterApply ...
func (a *Agent) UnRegisterApply(name string) error {
	return a.registry.UnRegisterApply(name)
}

// GetApplyFunc ...
func (a *Agent) GetApplyFunc(name string) (dbApi.ApplyFunc, bool, error) {
	return a.registry.GetApplyFunc(name)
}

// Apply ...
func (a *Agent) Apply(
	roLevel rkvApi.ConsistencyLevel,
	fn string,
	args []byte) (interface{}, error) {
	return a.raftDb.ApplyFunc(roLevel, fn, args)
}
