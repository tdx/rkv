package api

import "io"

// Backuper interface to backup backend
type Backuper interface {
	Backup(io.Writer) error
}
