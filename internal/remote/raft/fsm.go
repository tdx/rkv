package raft

import (
	"errors"
	"fmt"
	"io"

	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

var _ raft.FSM = (*fsm)(nil)

// raft FSM
type fsm struct {
	id     string
	db     dbApi.Backend
	logger log.Logger
}

// Apply will apply a log to the FSM. This is called from the raft library.
func (f *fsm) Apply(r *raft.Log) interface{} {
	f.logger.Trace("apply", "index:", r.Index, "term:", r.Term,
		"type:", r.Type, "data:", r.Data)
	switch r.Type {
	case raft.LogCommand:
		var req rpcRaft.LogData
		buf := r.Data
		if err := req.Unmarshal(buf); err != nil {
			return err
		}
		return f.applyData(req)

	case raft.LogConfiguration:
		return errors.New("raft.LogConfiguration does not emplemented in FSM")
	}
	return nil
}

// Snapshot
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.logger.Debug("snapshot")
	return &snapshot{
		id:     f.id,
		db:     f.db,
		logger: f.logger.Named("snapshot"),
	}, nil
}

// Restore
func (f *fsm) Restore(r io.ReadCloser) error {
	f.logger.Debug("restore")

	return f.db.Restore(r)
}

//
func (f *fsm) applyData(req rpcRaft.LogData) error {
	var err error
	for _, cmd := range req.Operations {
		switch cmd.OpType {
		case putOp:
			err = f.db.Put(cmd.Tab, cmd.Key, cmd.Val)
		case deleteOp:
			err = f.db.Delete(cmd.Tab, cmd.Key)
		default:
			err = fmt.Errorf("%q is not supported operation", cmd.OpType)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
