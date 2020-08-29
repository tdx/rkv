package raft

import (
	"net"
	"time"

	"github.com/tdx/raft"
)

var _ raft.StreamLayer = (*StreamLayer)(nil)

// StreamLayer is the network service provided to Raft
type StreamLayer struct {
	ln net.Listener
}

// NewStreamLayer ...
func NewStreamLayer(ln net.Listener) *StreamLayer {
	return &StreamLayer{
		ln: ln,
	}
}

// Dial creates a new network connection.
func (t *StreamLayer) Dial(
	addr raft.ServerAddress,
	timeout time.Duration) (net.Conn, error) {

	dialer := &net.Dialer{Timeout: timeout}

	return dialer.Dial("tcp", string(addr))
}

// Accept waits for the next connection.
func (t *StreamLayer) Accept() (net.Conn, error) {
	return t.ln.Accept()
}

// Close closes the StreamLayer
func (t *StreamLayer) Close() error {
	return t.ln.Close()
}

// Addr returns the binding address of the StreamLayer.
func (t *StreamLayer) Addr() net.Addr {
	return t.ln.Addr()
}
