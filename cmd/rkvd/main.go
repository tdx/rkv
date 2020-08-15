package main

import (
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/tdx/rkv"
	rkvApi "github.com/tdx/rkv/api"
	"github.com/tdx/rkv/db/bitcask"
	"github.com/tdx/rkv/db/bolt"
	"github.com/tdx/rkv/db/gmap"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	cli := &cli{}

	cmd := &cobra.Command{
		Use:     "rkvd",
		PreRunE: cli.setupConfig,
		RunE:    cli.run,
	}

	if err := setupFlags(cmd); err != nil {
		panic(err)
	}

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

type cli struct {
	rkvApi.Config
}

func setupFlags(cmd *cobra.Command) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	dataDir := path.Join(os.TempDir(), "rkvd")

	cmd.Flags().String("config-file", "", "Path to config file.")
	cmd.Flags().String("node-name", hostname, "Unique server ID.")
	cmd.Flags().String("data-dir", dataDir, "Directory to store DB, log and Raft data.")
	cmd.Flags().String("db", "bolt", "Database. Awaylable: bolt, map, bitcask.")
	cmd.Flags().Int("bitcask-max-data-file-size", (1 << 20), "Max data file size for bitcask.")
	cmd.Flags().String("discovery-addr", "0.0.0.0:8400", "Address to bind Serf on.")
	cmd.Flags().StringSlice("discovery-join-addrs", nil, "Serf addresses to join.")
	cmd.Flags().Int("raft-port", 8401, "Port for Raft connections.")
	cmd.Flags().String("raft-election-timeout", "1000ms", "Raft election timeout.")
	cmd.Flags().String("raft-commit-timeout", "50ms", "Raft commit timeout.")
	cmd.Flags().String("raft-heartbeat-timeout", "1000ms", "Raft heartbeat timeout.")
	cmd.Flags().String("raft-leader-lease-timeout", "500ms", "Raft leader lease timeout.")
	cmd.Flags().String("raft-snapshot-interval", "900s", "Raft snapshot interval.")
	cmd.Flags().Int("raft-snapshot-threshold", 8192, "Raft snapshot threshold.")
	cmd.Flags().Int("rpc-port", 8402, "Port for RPC connections.")
	cmd.Flags().String("http-addr", "0.0.0.0:8403", "Address to bind HTTP on.")
	cmd.Flags().String("log-level", "info", "Log level.")

	return viper.BindPFlags(cmd.Flags())
}

func (c *cli) setupConfig(cmd *cobra.Command, args []string) error {
	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		return err
	}
	viper.SetConfigFile(configFile)

	if err = viper.ReadInConfig(); err != nil {
		// it's ok if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// db
	dataDir := viper.GetString("data-dir")
	dbType := viper.GetString("db")
	switch dbType {
	case "map":
		c.Config.Backend, err = gmap.New(dataDir)
	case "bitcask":
		max := viper.GetInt("bitcask-max-data-file-size")
		c.Config.Backend, err = bitcask.New(dataDir, max)
	default:
		c.Config.Backend, err = bolt.New(dataDir)
	}
	if err != nil {
		return err
	}

	c.Config.NodeName = viper.GetString("node-name")
	c.Config.LogLevel = viper.GetString("log-level")
	c.Config.DiscoveryAddr = viper.GetString("discovery-addr")
	c.Config.DiscoveryJoinAddrs = viper.GetStringSlice("discovery-join-addrs")

	c.Config.Raft.CommitTimeout = viper.GetDuration("raft-commit-timeout")
	c.Config.Raft.ElectionTimeout = viper.GetDuration("raft-election-timeout")
	c.Config.Raft.HeartbeatTimeout = viper.GetDuration("raft-heartbeat-timeout")
	c.Config.Raft.LeaderLeaseTimeout = viper.GetDuration("raft-leader-lease-timeout")
	c.Config.Raft.SnapshotInterval = viper.GetDuration("raft-snapshot-interval")
	c.Config.Raft.SnapshotThreshold = uint64(viper.GetInt("raft-snapshot-threshold"))

	c.Config.RaftPort = viper.GetInt("raft-port")
	c.Config.RPCPort = viper.GetInt("rpc-port")
	c.Config.HTTPAddr = viper.GetString("http-addr")

	return nil
}

func (c *cli) run(cmd *cobra.Command, args []string) error {
	client, err := rkv.NewClient(&c.Config)
	if err != nil {
		return err
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc

	return client.Shutdown()
}
