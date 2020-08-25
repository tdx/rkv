package route

import (
	"context"
	"fmt"
	"sync"

	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	log "github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

//
// route write requests to leader
//

// Resolver ...
type Resolver struct {
	name           string
	logger         log.Logger
	mu             sync.Mutex
	clientConn     resolver.ClientConn // from client
	resolverConn   *grpc.ClientConn    // to server
	serviceConfig  *serviceconfig.ParseResult
	targetEndpoint string
}

//
// Builder
//
var _ resolver.Builder = (*Resolver)(nil)

// Build builds gRPC Resolver
func (r *Resolver) Build(
	target resolver.Target,
	cc resolver.ClientConn,
	opts resolver.BuildOptions) (resolver.Resolver, error) {

	r.logger.Debug("Build", "target", target.Endpoint)

	r.clientConn = cc
	r.targetEndpoint = target.Endpoint

	r.serviceConfig = r.clientConn.ParseServiceConfig(
		fmt.Sprintf(`{"loadBalancingConfig":[{"%s":{}}]}`, r.name),
	)

	var (
		err      error
		dialOpts []grpc.DialOption = []grpc.DialOption{grpc.WithInsecure()}
	)

	r.resolverConn, err = grpc.Dial(target.Endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	r.ResolveNow(resolver.ResolveNowOptions{})

	return r, nil
}

// Scheme returns resolver scheme
func (r *Resolver) Scheme() string {
	return r.name
}

//
// Resolver
//

var _ resolver.Resolver = (*Resolver)(nil)

// ResolveNow ...
func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {

	r.mu.Lock()
	defer r.mu.Unlock()

	client := rpcApi.NewStorageClient(r.resolverConn)
	ctx := context.Background()
	res, err := client.Servers(ctx, &rpcApi.ServersArgs{})
	if err != nil {
		r.logger.Error("ResolveNow", "target", r.targetEndpoint, "error", err)
		return
	}

	var (
		leader string
		addrs  []resolver.Address
	)
	for i := range res.Servers {
		server := res.Servers[i]
		if !server.IsLeader {
			continue
		}
		leader = server.Host + ":" + server.RpcPort
		addrs = append(addrs, resolver.Address{
			Addr: leader,
			Attributes: attributes.New(
				"is_leader", server.IsLeader,
				"rpc_addr", leader,
			),
		})
	}

	r.logger.Debug("ResolveNow", "target", r.targetEndpoint,
		"leader-addr", leader)

	r.clientConn.UpdateState(resolver.State{
		Addresses:     addrs,
		ServiceConfig: r.serviceConfig,
	})
}

// Close ...
func (r *Resolver) Close() {

	r.logger.Debug("Close", "target", r.targetEndpoint)

	if err := r.resolverConn.Close(); err != nil {
		r.logger.Error("Close", "target", r.targetEndpoint, "error", err)
	}
}
