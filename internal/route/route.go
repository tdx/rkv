package route

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

// Route ...
type Route struct {
	*Config
	resolver *Resolver
	picker   *Picker
}

// New returns Route instance
func New(config *Config) *Route {
	
	s := &Route{
		Config: config,
		resolver: &Resolver{
			name:   config.Name,
			logger: config.Logger.Named("resolver"),
		},
		picker: &Picker{
			logger: config.Logger.Named("picker"),
		},
	}

	resolver.Register(s.resolver)

	balancer.Register(
		base.NewBalancerBuilder(config.Name, s.picker, base.Config{}),
	)

	return s
}

// Name returns router name
func (r *Route) Name() string {
	return r.Config.Name
}
