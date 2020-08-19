package raft

import (
	"errors"

	dbApi "github.com/tdx/rkv/db/api"
)

func (f *fsm) applyFunc(name, args []byte) interface{} {

	dbApp, ok := f.db.(dbApi.Applier)
	if !ok {
		return errors.New("db does not implement db/api/Apply")
	}

	fn, ro, err := f.appReg.GetApplyFunc(string(name))
	if err != nil {
		return err
	}

	val, err := dbApp.Apply(fn, args, ro)
	if err != nil {
		return err
	}

	return val
}
