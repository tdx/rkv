package raft

import (
	"errors"

	dbApi "github.com/tdx/rkv/db/api"
)

func (f *fsm) applyFunc(name []byte, args ...[]byte) interface{} {

	dbApp, ok := f.db.(dbApi.Applier)
	if !ok {
		return errors.New("db does not implement db/api/Apply")
	}

	fn, ro, err := f.appReg.GetApplyFunc(string(name))
	if err != nil {
		return err
	}

	var val interface{}
	if ro {
		val, err = dbApp.ApplyRead(fn, args...)
	} else {
		val, err = dbApp.ApplyWrite(fn, args...)
	}
	if err != nil {
		return err
	}

	return val
}
