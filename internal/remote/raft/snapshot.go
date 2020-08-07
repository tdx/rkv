package raft

import (
	dbApi "github.com/tdx/rkv/internal/db/api"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	id     string
	db     dbApi.Backuper
	logger log.Logger
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	s.logger.Debug("persist", "id", sink.ID())

	if err := s.db.Backup(sink); err != nil {
		_ = sink.Cancel()
		return err
	}

	return sink.Close()
}

func (s *snapshot) Release() {}
