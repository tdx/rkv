package api

import (
	"errors"

	dbApi "github.com/tdx/rkv/db/api"
)

// Errors
var (
	ErrApplyFuncRegistered   = errors.New("apply func already registered")
	ErrApplyFuncDoesNotExist = errors.New("apply func does not exist")
)

// ApplyRegistrator is an interface to reg/unreg function to work with db
type ApplyRegistrator interface {
	RegisterApplyWrite(name string, fn dbApi.ApplyFunc) error
	RegisterApplyRead(name string, fn dbApi.ApplyFunc) error
	UnRegisterApply(name string) error
	GetApplyFunc(name string) (dbApi.ApplyFunc, bool, error)
}
