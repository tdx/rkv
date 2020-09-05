package discovery

import (
	"fmt"
	"net"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/serf/serf"
)

// Membership ...
type Membership struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
	persist *persist
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

	m.persist, err = newPersist(m.DataDir)
	if err != nil {
		return err
	}

	logger := m.Config.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  fmt.Sprintf("serf-%s", m.Config.NodeName),
			Level: log.Debug,
		})
	}
	m.Config.Logger = logger

	if len(m.StartJoinAddrs) < 2 {
		if v := m.persist.GetJoins(); v != "" {
			joinAddrs := strings.Split(v, ",")
			if len(joinAddrs) > len(m.StartJoinAddrs) {
				m.StartJoinAddrs = joinAddrs
			}
		}
	}

	logger.Debug("setup", "join-addrs", m.StartJoinAddrs)

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

	config.LogOutput = logger.StandardLogger(&log.StandardLoggerOptions{
		InferLevels: true,
	}).Writer()
	config.MemberlistConfig.LogOutput = config.LogOutput

	m.serf, err = serf.Create(config)
	if err != nil {
		return err
	}
	go m.eventHandler()
	if len(m.StartJoinAddrs) > 0 {
		if _, err = m.serf.Join(m.StartJoinAddrs, true); err != nil {
			logger.Error("initial join failed", "error", err)
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
		m.Config.Logger.Error("JOIN",
			"id", member.Name,
			"address", member.Tags["raft_addr"],
			"error", err,
		)
	}
	if err := m.persistsMembers(); err != nil {
		m.Config.Logger.Error("persist members failed", "error", err)
	}
}

func (m *Membership) handleLeave(member serf.Member) {
	err := m.handler.Leave(member.Name, member.Tags, m.isLocal(member))
	if err != nil {
		m.Config.Logger.Error("LEAVE",
			"id", member.Name,
			"address", member.Tags["raft_addr"],
			"error", err,
		)
	}
	if err = m.persistsMembers(); err != nil {
		m.Config.Logger.Error("persist members failed", "error", err)
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
	m.Config.Logger.Trace("stopping")
	return m.serf.Leave()
}

func (m *Membership) persistsMembers() error {
	var (
		serfAddrs []string
		localName = m.serf.LocalMember().Name
	)
	for _, member := range m.serf.Members() {
		if localName != member.Name {
			serfAddr, ok := member.Tags["serf_addr"]
			if ok {
				addr, err := normaliseAddr(serfAddr)
				if err != nil {
					m.Config.Logger.Error("persist Members", "id", member.Name,
						"serf-addr", serfAddr, "error", err)
					continue
				}
				serfAddrs = append(serfAddrs, addr)
			}
		}
	}
	if len(serfAddrs) == 0 {
		return nil
	}

	return m.persist.SaveJoins(strings.Join(serfAddrs, ","))
}

func normaliseAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, err
	}
	switch host {
	case "::":
		host = "ip6-localhost"
	case "", ":":
		host = "localhost"
	}
	return net.JoinHostPort(host, port), nil
}

func (m *Membership) logLevel() log.Level {
	if m.Config.Logger.IsTrace() {
		return log.Trace
	} else if m.Config.Logger.IsDebug() {
		return log.Debug
	} else if m.Config.Logger.IsInfo() {
		return log.Info
	} else if m.Config.Logger.IsWarn() {
		return log.Warn
	} else {
		return log.Error
	}
}
