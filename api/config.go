package api

// Config to create client
type Config struct {
	NodeName string
	// Membership ip:port used by Serf to discover nodes and create cluster
	DiscoveryAddr string
	// DiscoveryJoinAddrs is empty for leader
	// For followers contains leader address [+running followers addresses]
	DiscoveryJoinAddrs []string
	// Distributed database port
	DistributedPort int
	// LogLevel: error | warn | info | debug | trace
	LogLevel string
	// Database directory
	DataDir string
}
