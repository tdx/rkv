package raft

import (
	"encoding/gob"
	"errors"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"
)

// ApplyFuncArgs used in gob enc/dec
type ApplyFuncArgs struct {
	Args interface{}
}

func init() {
	gob.Register(ApplyFuncArgs{})
}

// ApplyFunc calls 'name' function on raft cluster
func (d *Backend) ApplyFunc(
	roLevel rkvApi.ConsistencyLevel,
	name string, args []byte) (interface{}, error) {

	// check func exists
	fn, ro, err := d.fsm.appReg.GetApplyFunc(name)
	if err != nil {
		return nil, err
	}

	// call on local node
	if ro {

		readLocal := true

		switch roLevel {
		case rkvApi.ReadLeader:
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

			resp, err := dbApp.Apply(fn, args, ro)
			if err != nil {
				return nil, err
			}

			return resp, nil
		}
	}

	// argsSerialized, err := i2b(&ApplyFuncArgs{args})
	// if err != nil {
	// 	return nil, err
	// }

	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: applyOp,
				Tab:    []byte(name),
				Key:    args,
			},
		},
	}

	return d.applyLog(req)
}

// func i2b(args *ApplyFuncArgs) ([]byte, error) {
// 	var buf bytes.Buffer
// 	enc := gob.NewEncoder(&buf)
// 	if err := enc.Encode(args); err != nil {
// 		return nil, fmt.Errorf("i2b failed: %v", err)
// 	}
// 	return buf.Bytes(), nil
// }

// func b2i(buf []byte) (interface{}, error) {
// 	var v ApplyFuncArgs
// 	dec := gob.NewDecoder(bytes.NewReader(buf))
// 	if err := dec.Decode(&v); err != nil {
// 		return nil, fmt.Errorf("b2i failed: %v", err)
// 	}

// 	return v.Args, nil
// }
