package registry

import (
	"sync"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
)

type afn struct {
	fn dbApi.ApplyFunc
	ro bool
}

type reg struct {
	funcs map[string]*afn
	muf   sync.RWMutex
}

var _ rkvApi.ApplyRegistrator = (*reg)(nil)

// NewApplyRegistrator ...
func NewApplyRegistrator() rkvApi.ApplyRegistrator {
	return &reg{
		funcs: make(map[string]*afn),
	}
}

// RegisterApply ...
func (r *reg) RegisterApply(name string, fn dbApi.ApplyFunc, ro bool) error {
	r.muf.RLock()
	_, ok := r.funcs[name]
	r.muf.RUnlock()

	if ok {
		return rkvApi.ErrApplyFuncRegistered
	}

	r.muf.Lock()
	r.funcs[name] = &afn{fn, ro}
	r.muf.Unlock()

	return nil
}

// UnRegisterApply ...
func (r *reg) UnRegisterApply(name string) error {
	r.muf.RLock()
	_, ok := r.funcs[name]
	r.muf.RUnlock()

	if !ok {
		return rkvApi.ErrApplyFuncDoesNotExist
	}

	r.muf.Lock()
	delete(r.funcs, name)
	r.muf.Unlock()

	return nil
}

// GetApplyFunc ...
func (r *reg) GetApplyFunc(name string) (dbApi.ApplyFunc, bool, error) {
	r.muf.RLock()
	v, ok := r.funcs[name]
	r.muf.RUnlock()

	if !ok {
		return nil, false, rkvApi.ErrApplyFuncDoesNotExist
	}

	return v.fn, v.ro, nil
}
