package route

import (
	"sync"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

// Picker ...
type Picker struct {
	logger     log.Logger
	mu         sync.RWMutex
	leader     balancer.SubConn
	leaderAddr string
	current    uint64
}

//
// Builder
//
var _ base.PickerBuilder = (*Picker)(nil)

// Build ...
func (p *Picker) Build(buildInfo base.PickerBuildInfo) balancer.Picker {
	p.mu.Lock()
	defer p.mu.Unlock()

	for sc, scInfo := range buildInfo.ReadySCs {
		isLeader := scInfo.Address.Attributes.Value("is_leader").(bool)
		if isLeader {
			p.leader = sc
			p.leaderAddr = scInfo.Address.Attributes.Value("rpc_addr").(string)
		}
	}

	p.logger.Debug("Build", "leader", p.leaderAddr)

	return p
}

//
// Picker
//
var _ balancer.Picker = (*Picker)(nil)

// Pick ...
func (p *Picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {

	p.logger.Debug("Pick", "leader", p.leaderAddr,
		"method", info.FullMethodName)

	p.mu.RLock()
	defer p.mu.RUnlock()

	var result balancer.PickResult

	// always leader
	result.SubConn = p.leader

	return result, nil
}
