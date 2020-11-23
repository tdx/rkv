package raft

import (
	"errors"
	"fmt"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"
)

// ApplyFuncRead calls 'name' function on raft cluster
func (d *Backend) ApplyFuncRead(
	roLevel rkvApi.ConsistencyLevel,
	name string, args ...[]byte) (interface{}, error) {

	// check func exists
	fn, ro, err := d.fsm.appReg.GetApplyFunc(name)
	if err != nil {
		return nil, err
	}

	if ro != true {
		return nil, fmt.Errorf("apply func '%s' registered as write func", name)
	}

	readLocal := true

	switch roLevel {
	case rkvApi.ReadLeader:
		// TODO: read from d.leader
		if !d.IsLeader() {
			return nil, rkvApi.ErrNodeIsNotALeader
		}
	case rkvApi.ReadCluster:
		readLocal = false
	}

	if readLocal {
		dbApp, ok := d.fsm.db.(dbApi.Applier)
		if !ok {
			return nil, errors.New("db does not implement db/api/Apply")
		}

		resp, err := dbApp.ApplyRead(fn, args...)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: applyOp,
				Tab:    []byte(name),
				Args:   args,
			},
		},
	}

	return d.applyLog(req)
}

// ApplyFuncWrite calls 'name' function on raft cluster
func (d *Backend) ApplyFuncWrite(
	name string, args ...[]byte) (interface{}, error) {

	// check func exists
	_, _, err := d.fsm.appReg.GetApplyFunc(name)
	if err != nil {
		return nil, err
	}

	if !d.IsLeader() {
		// TODO: read from d.leader
		return nil, rkvApi.ErrNodeIsNotALeader
	}

	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: applyOp,
				Tab:    []byte(name),
				Args:   args,
			},
		},
	}

	return d.applyLog(req)
}
