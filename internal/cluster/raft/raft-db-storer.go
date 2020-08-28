package raft

import (
	"context"
	"fmt"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	putOp uint32 = 1 << iota
	deleteOp
	getOp
	applyOp
)

//
// Get returns value from local backend
//
func (d *Backend) Get(
	lvl rkvApi.ConsistencyLevel, tab, key []byte) ([]byte, error) {

	doLeaderRPC := false

	switch lvl {
	case rkvApi.ReadAny:
		return d.fsm.Get(tab, key)

	case rkvApi.ReadLeader:
		if d.IsLeader() {
			return d.fsm.Get(tab, key)
		}

		if d.leaderConn == nil {
			return nil, rkvApi.ErrNodeIsNotALeader
		}

		doLeaderRPC = true

	case rkvApi.ReadCluster:
		if !d.IsLeader() {
			if d.leaderConn == nil {
				return nil, rkvApi.ErrNodeIsNotALeader
			}
			doLeaderRPC = true
		}
	}

	if doLeaderRPC {
		resp, err := d.leaderConn.Get(
			context.Background(),
			&rpcApi.StorageGetArgs{
				Tab: tab,
				Key: key,
			})
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				return nil, rkvApi.ErrNodeIsNotALeader
			case codes.NotFound:
				return nil, dbApi.ErrNoKey(key)
			default:
				d.logger.Error("grpc unexpected error", "code", st.Code(),
					"message", st.Message())
				return nil, rkvApi.ErrInternalError
			}
		}
		return resp.Val, nil
	}

	// raft read from cluster
	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: getOp,
				Tab:    tab,
				Key:    key,
			},
		},
	}

	res, err := d.applyLog(req)
	if err != nil {
		return nil, err
	}

	switch res := res.(type) {
	case []byte:
		return res, nil
	case []interface{}:
		switch res := res[0].(type) {
		case []byte:
			return res, nil
		case error:
			return nil, res
		default:
			return nil,
				fmt.Errorf("raft.Get: unexprected result: %v(%T)", res, res)
		}
	case error:
		return nil, res
	default:
		return nil,
			fmt.Errorf("raft.Get: unexprected result: %v(%T)", res, res)
	}
}

// Put apply value via raft
func (d *Backend) Put(tab, key, val []byte) error {

	if !d.IsLeader() {
		if d.leaderConn == nil {
			return rkvApi.ErrNodeIsNotALeader
		}

		// do leader RPC
		_, err := d.leaderConn.Put(
			context.Background(),
			&rpcApi.StoragePutArgs{
				Tab: tab,
				Key: key,
				Val: val,
			})
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				return rkvApi.ErrNodeIsNotALeader
			}
			return err
		}
		return nil
	}

	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: putOp,
				Tab:    tab,
				Key:    key,
				Val:    val,
			},
		},
	}

	_, err := d.applyLog(req)

	return err
}

// Delete inserts an entry in the log to delete the given key
func (d *Backend) Delete(tab, key []byte) error {

	if !d.IsLeader() {
		if d.leaderConn == nil {
			return rkvApi.ErrNodeIsNotALeader
		}

		// do leader RPC
		_, err := d.leaderConn.Delete(
			context.Background(),
			&rpcApi.StorageDeleteArgs{
				Tab: tab,
				Key: key,
			})
		if err != nil {
			st := status.Convert(err)
			switch st.Code() {
			case codes.Unavailable:
				return rkvApi.ErrNodeIsNotALeader
			case codes.NotFound:
				return dbApi.ErrNoKey(key)
			default:
				d.logger.Error("grpc unexpected error", "code", st.Code(),
					"message", st.Message())
				return rkvApi.ErrInternalError
			}
		}
		return err
	}

	req := &rpcRaft.LogData{
		Operations: []*rpcRaft.LogOperation{
			{
				OpType: deleteOp,
				Tab:    tab,
				Key:    key,
			},
		},
	}

	_, err := d.applyLog(req)

	return err
}

// Batch apply multiple put, delete operations
func (d *Backend) Batch(commands []*dbApi.BatchEntry) (interface{}, error) {
	ops := make([]*rpcRaft.LogOperation, 0, len(commands))
	for _, cmd := range commands {
		switch cmd.Operation {
		case dbApi.PutOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: putOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Entry.Key,
				Val:    cmd.Entry.Val,
			})
		case dbApi.DeleteOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: deleteOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Entry.Key,
			})
		case dbApi.GetOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: getOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Entry.Key,
			})
		case dbApi.ApplyOperation:
			ops = append(ops, &rpcRaft.LogOperation{
				OpType: applyOp,
				Tab:    cmd.Entry.Tab,
				Key:    cmd.Apply.Args,
			})
		}
	}

	req := &rpcRaft.LogData{
		Operations: ops,
	}

	return d.applyLog(req)
}
