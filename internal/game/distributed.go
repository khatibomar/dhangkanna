package game

import (
	"fmt"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	api "github.com/khatibomar/dhangkanna/cmd/api/v1"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Raft struct {
		raft.Config
		BindAddr    string
		StreamLayer *StreamLayer
		Bootstrap   bool
	}
}

type DistributedGame struct {
	*Game
	config Config
	Raft   *raft.Raft
	logger *log.Logger
}

func NewDistributedGame(dataDir string, config Config) (*DistributedGame, error) {
	g := &DistributedGame{
		config: config,
		logger: log.New(os.Stdout, "distributed game: ", log.LstdFlags|log.Lshortfile),
	}
	g.Game = New()

	if err := g.setupRaft(dataDir); err != nil {
		return nil, err
	}
	g.logger.Println("DistributedGame initialized successfully")
	return g, nil
}

func (g *DistributedGame) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("timed out")
		case <-ticker.C:
			if l, _ := g.Raft.LeaderWithID(); l != "" {
				g.logger.Printf("Leader found: %s\n", l)
				return nil
			}
		}
	}
}

func (g *DistributedGame) Join(id, addr string) error {
	g.logger.Printf("Joining the cluster with ID: %s and address: %s\n", id, addr)
	configFuture := g.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return err
	}
	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)
	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				g.logger.Printf("Server ( %s , %s ) already in the cluster\n", srv.ID, srv.Address)
				return nil
			}
			g.logger.Printf("Removing server from the cluster: %s\n", srv.ID)

			removeFuture := g.Raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				g.logger.Printf("Error removing server from the cluster: %s\n", err)

				return err
			}
		}
	}
	g.logger.Printf("Adding server to the cluster: %s\n", id)
	addFuture := g.Raft.AddVoter(serverID, serverAddr, 0, 0)
	if addFuture.Error() != nil {
		return addFuture.Error()
	}
	g.logger.Printf("Server ( %s , %s ) joined the cluster successfully\n", serverID, serverAddr)

	return nil
}

func (g *DistributedGame) Leave(id string) error {
	removeFuture := g.Raft.RemoveServer(raft.ServerID(id), 0, 0)
	return removeFuture.Error()
}

func (g *DistributedGame) GetServers() ([]*api.Server, error) {
	future := g.Raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}
	var servers []*api.Server
	leaderAdrr, _ := g.Raft.LeaderWithID()
	for _, server := range future.Configuration().Servers {
		servers = append(servers, &api.Server{
			Id:       string(server.ID),
			RpcAddr:  string(server.Address),
			IsLeader: leaderAdrr == server.Address,
		})
	}
	return servers, nil
}

func (g *DistributedGame) Close() error {
	f := g.Raft.Shutdown()
	return f.Error()
}

func (g *DistributedGame) setupRaft(dataDir string) error {
	g.logger.Println("setting up raft")

	fsm := fsm{game: g.Game}

	logDir := filepath.Join(dataDir, "raft")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	stableStore, err := raftboltdb.NewBoltStore(
		filepath.Join(dataDir, "raft", "stable"),
	)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.NewBoltStore(
		filepath.Join(dataDir, "raft", "store"),
	)
	if err != nil {
		return err
	}

	retain := 1
	snapshotStore, err := raft.NewFileSnapshotStore(
		filepath.Join(dataDir, "raft", "log"),
		retain,
		os.Stderr,
	)
	if err != nil {
		return err
	}

	maxPool := 5
	timeout := 10 * time.Second
	transport := raft.NewNetworkTransport(
		g.config.Raft.StreamLayer,
		maxPool,
		timeout,
		os.Stderr,
	)

	config := raft.DefaultConfig()
	config.LocalID = g.config.Raft.LocalID
	if g.config.Raft.HeartbeatTimeout != 0 {
		config.HeartbeatTimeout = g.config.Raft.HeartbeatTimeout
	}
	if g.config.Raft.ElectionTimeout != 0 {
		config.ElectionTimeout = g.config.Raft.ElectionTimeout
	}
	if g.config.Raft.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = g.config.Raft.LeaderLeaseTimeout
	}
	if g.config.Raft.CommitTimeout != 0 {
		config.CommitTimeout = g.config.Raft.CommitTimeout
	}

	g.Raft, err = raft.NewRaft(
		config,
		fsm,
		logStore,
		stableStore,
		snapshotStore,
		transport,
	)
	if err != nil {
		return err
	}
	hasState, err := raft.HasExistingState(
		logStore,
		stableStore,
		snapshotStore,
	)
	if err != nil {
		return err
	}
	if g.config.Raft.Bootstrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(g.config.Raft.BindAddr),
			}},
		}
		err = g.Raft.BootstrapCluster(config).Error()
	}
	g.logger.Println("Done setting up raft")
	return err
}
