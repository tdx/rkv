package agent

import (
	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
)

var _ rkvApi.ApplyRegistrator = (*Agent)(nil)

// RegisterApply ...
func (a *Agent) RegisterApply(name string, fn dbApi.ApplyFunc, ro bool) error {
	return a.registry.RegisterApply(name, fn, ro)
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
