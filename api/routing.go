package api

// RoutingPolicy type
type RoutingPolicy int

// Available routing policies for mutate state operations
const (
	RouteToLeader       RoutingPolicy = 0
	RouteReturnRedirect RoutingPolicy = 1
)
