package raft

import (
	"errors"
	"fmt"
	"io"

	rkvApi "github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	rpcRaft "github.com/tdx/rkv/internal/rpc/raft"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

var _ raft.FSM = (*fsm)(nil)

// var _ raft.BatchingFSM = (*fsm)(nil)

// raft FSM
type fsm struct {
	id     string
	db     dbApi.Backend
	logger log.Logger
	appReg rkvApi.ApplyRegistrator
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

// ApplyBatch will apply a set of logs to the FSM. This is called from the raft
// library.
func (f *fsm) ApplyBatch(logs []*raft.Log) []interface{} {

	f.logger.Trace("applyBatch", "logs", len(logs))

	if len(logs) == 0 {
		return []interface{}{}
	}

	readOnly := true
	commands := make([][]*dbApi.BatchEntry, len(logs))

	for i, log := range logs {

		switch log.Type {
		case raft.LogCommand:

			var command rpcRaft.LogData
			if err := command.Unmarshal(log.Data); err != nil {
				f.logger.Error("error proto unmarshaling log data", "error", err)
				panic("error proto unmarshaling log data")
			}
			for _, cmd := range command.Operations {

				f.logger.Trace("applyBatch", "raft_index", log.Index,
					"raft_term", log.Term, "cmd_type", cmd.OpType,
					"log", i, "commands", len(command.Operations))

				switch cmd.OpType {
				case putOp:
					readOnly = false
					commands[i] = append(commands[i], &dbApi.BatchEntry{
						Operation: dbApi.PutOperation,
						Entry: &dbApi.Entry{
							Tab: cmd.Tab,
							Key: cmd.Key,
							Val: cmd.Val,
						}})

				case deleteOp:
					readOnly = false
					commands[i] = append(commands[i], &dbApi.BatchEntry{
						Operation: dbApi.DeleteOperation,
						Entry: &dbApi.Entry{
							Tab: cmd.Tab,
							Key: cmd.Key,
						}})

				case getOp:
					commands[i] = append(commands[i], &dbApi.BatchEntry{
						Operation: dbApi.GetOperation,
						Entry: &dbApi.Entry{
							Tab: cmd.Tab,
							Key: cmd.Key,
						}})

				case applyOp:
					fnName := string(cmd.Tab)
					fn, ro, err := f.appReg.GetApplyFunc(fnName)
					if !ro {
						readOnly = false
					}

					command := &dbApi.BatchEntry{
						Operation: dbApi.ApplyOperation,
						Apply: &dbApi.Apply{
							Fn:       fn,
							Args:     cmd.Args,
							ReadOnly: ro,
						}}

					if err != nil {
						f.logger.Error("applyFun", "func", fnName,
							"args", len(cmd.Args), "error", err)
						command.Result = err
					}
					commands[i] = append(commands[i], command)

				default:
					f.logger.Error("applyBatch", "bad_cmd", cmd.OpType)
					continue
				}
			}
		}
	}

	f.logger.Trace("applyBatch", "commands:", len(commands), "ro", readOnly)

	if len(commands) > 0 {
		err := f.db.Batch(commands, readOnly)
		if err != nil {
			f.logger.Error("Batch failed", "error", err)
		}
	}

	// Build the responses. The logs array is used here to ensure we reply to
	// all command values; even if they are not of the types we expect. This
	// should future proof this function from more log types being provided.
	resp := make([]interface{}, len(logs))
	for i := range logs {
		if len(commands) > 0 {
			if len(commands[i]) == 1 {
				resp[i] = commands[i][0].Result
				// f.logger.Trace("ApplyBatch", "log", i, "response", resp[i])
				continue
			}
			cmdResp := make([]interface{}, 0, len(commands[i]))
			for _, cmd := range commands[i] {
				cmdResp = append(cmdResp, cmd.Result)
			}
			resp[i] = cmdResp
			// f.logger.Trace("ApplyBatch", "log", i, "response", cmdResp)
			continue
		}
		resp[i] = true
	}

	// f.logger.Trace("ApplyBatch", "logs", len(logs), "response", resp)

	return resp
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
func (f *fsm) applyData(req rpcRaft.LogData) interface{} {
	var err error
	for _, cmd := range req.Operations {
		switch cmd.OpType {
		case putOp:
			err = f.db.Put(cmd.Tab, cmd.Key, cmd.Val)
		case deleteOp:
			err = f.db.Delete(cmd.Tab, cmd.Key)
		case getOp:
			val, err := f.db.Get(cmd.Tab, cmd.Key)
			if err != nil {
				return err
			}
			return val
		case applyOp:
			return f.applyFunc(cmd.Tab, cmd.Args...)
		default:
			err = fmt.Errorf("%q is not supported operation", cmd.OpType)
		}

		if err != nil {
			return err
		}
	}

	return nil
}
