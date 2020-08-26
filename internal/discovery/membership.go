package discovery

import (
	"fmt"
	"net"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/serf/serf"
)

// Membership ...
type Membership struct {
	Config
	logger  log.Logger
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
}

// New ...
func New(handler Handler, config Config) (*Membership, error) {

	m := &Membership{
		Config:  config,
		handler: handler,
	}
	if err := m.setupSerf(); err != nil {
		return nil, err
	}
	return m, nil
}

// Join joins node to serf cluster
func (m *Membership) Join(existing []string) (int, error) {
	return m.serf.Join(existing, true)
}

func (m *Membership) setupSerf() (err error) {
	logger := m.Config.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("serf-%s", m.Config.NodeName),
			Level: log.Debug,
		})
	}
	m.logger = logger

	addr, err := net.ResolveTCPAddr("tcp", m.BindAddr)
	if err != nil {
		return err
	}
	config := serf.DefaultConfig()
	config.Init()
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	m.events = make(chan serf.Event)
	config.EventCh = m.events
	config.Tags = m.Tags
	config.NodeName = m.Config.NodeName

	config.Logger = m.logger.StandardLogger(&log.StandardLoggerOptions{
		InferLevels: true,
	})

	m.serf, err = serf.Create(config)
	if err != nil {
		return err
	}
	go m.eventHandler()
	if len(m.StartJoinAddrs) > 0 {
		_, err = m.serf.Join(m.StartJoinAddrs, true)
		if err != nil {
			return err
		}
	}
	return nil
}

// Handler ...
type Handler interface {
	Join(name string, tags map[string]string, local bool) error
	Leave(name string, tags map[string]string, local bool) error
}

func (m *Membership) eventHandler() {
	for e := range m.events {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				m.handleJoin(member)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed, serf.EventMemberReap:
			for _, member := range e.(serf.MemberEvent).Members {
				m.handleLeave(member)
			}
		}
	}
}

func (m *Membership) handleJoin(member serf.Member) {
	tags := member.Tags
	tags["ip"] = member.Addr.String()
	if err := m.handler.Join(member.Name, tags, m.isLocal(member)); err != nil {
		m.logger.Error("JOIN",
			"id", member.Name,
			"address", member.Tags["raft_addr"],
			"error", err,
		)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	if err := m.handler.Leave(
		member.Name,
		member.Tags,
		m.isLocal(member),
	); err != nil {
		m.logger.Error("LEAVE",
			"id", member.Name,
			"address", member.Tags["raft_addr"],
			"error", err,
		)
	}
}

func (m *Membership) isLocal(member serf.Member) bool {
	return m.serf.LocalMember().Name == member.Name
}

// Members ...
func (m *Membership) Members() []serf.Member {
	return m.serf.Members()
}

// Leave ...
func (m *Membership) Leave() error {
	return m.serf.Leave()
}
