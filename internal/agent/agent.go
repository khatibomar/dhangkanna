package agent

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/raft"
	"github.com/khatibomar/dhangkanna/internal/discovery"
	"github.com/khatibomar/dhangkanna/internal/game"
	"github.com/khatibomar/dhangkanna/internal/server"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type Agent struct {
	Config          Config
	mux             cmux.CMux
	DistributedGame *game.DistributedGame
	server          *grpc.Server
	discovery       *discovery.Discovery
	logger          *log.Logger
	shutdown        bool
	shutdowns       chan struct{}
	shutdownLock    sync.Mutex
}

type Config struct {
	BindAddr       string
	RPCPort        int
	NodeName       string
	StartJoinAddrs []string
	Bootstrap      bool
	DataDir        string
}

func (c Config) RPCAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.RPCPort), nil
}

func New(config Config) (*Agent, error) {
	a := &Agent{
		Config:    config,
		shutdowns: make(chan struct{}),
		logger:    log.New(os.Stdout, "agent: ", log.LstdFlags|log.Lshortfile),
	}
	setup := []func() error{
		a.setupMux,
		a.setupGame,
		a.setupServer,
		a.setupDiscovery,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	go func() {
		if err := a.serve(); err != nil {
			a.logger.Printf("error while serving mux : %v", err)
		}
	}()

	return a, nil
}

func (a *Agent) serve() error {
	if err := a.mux.Serve(); err != nil {
		_ = a.Shutdown()
		return err
	}
	return nil
}

func (a *Agent) setupMux() error {
	addr, err := net.ResolveTCPAddr("tcp", a.Config.BindAddr)
	if err != nil {
		return err
	}
	rpcAddr := fmt.Sprintf(
		"%s:%d",
		addr.IP.String(),
		a.Config.RPCPort,
	)
	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		return err
	}
	a.mux = cmux.New(ln)
	return nil
}

func (a *Agent) setupGame() error {
	raftLn := a.mux.Match(func(reader io.Reader) bool {
		b := make([]byte, 1)
		if _, err := reader.Read(b); err != nil {
			return false
		}
		return bytes.Compare(b, []byte{byte(game.RaftRPC)}) == 0
	})
	gameConfig := game.Config{}
	gameConfig.Raft.StreamLayer = game.NewStreamLayer(
		raftLn,
	)
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		return err
	}
	gameConfig.Raft.BindAddr = rpcAddr
	gameConfig.Raft.LocalID = raft.ServerID(a.Config.NodeName)
	gameConfig.Raft.Bootstrap = a.Config.Bootstrap
	a.DistributedGame, err = game.NewDistributedGame(
		a.Config.DataDir,
		gameConfig,
	)
	if err != nil {
		return err
	}
	if a.Config.Bootstrap {
		err = a.DistributedGame.WaitForLeader(3 * time.Second)
	}
	return err
}

func (a *Agent) setupDiscovery() error {
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		a.logger.Printf("Error getting RPC address: %v", err)
		return err
	}
	a.discovery, err = discovery.New(a.DistributedGame, discovery.Config{
		NodeName: a.Config.NodeName,
		BindAddr: a.Config.BindAddr,
		Tags: map[string]string{
			"rpc_addr": rpcAddr,
		},
		StartJoinsAddresses: a.Config.StartJoinAddrs,
	})
	return err
}

func (a *Agent) setupServer() error {
	serverConfig := &server.Config{
		Game: a.DistributedGame.Game,
	}
	var opts []grpc.ServerOption
	var err error
	a.server, err = server.NewGRPCServer(serverConfig, opts...)
	if err != nil {
		return err
	}
	grpcLn := a.mux.Match(cmux.Any())
	go func() {
		if err := a.server.Serve(grpcLn); err != nil {
			_ = a.Shutdown()
		}
	}()
	return err
}

func (a *Agent) Shutdown() error {
	a.shutdownLock.Lock()
	defer a.shutdownLock.Unlock()
	if a.shutdown {
		return nil
	}
	a.shutdown = true
	close(a.shutdowns)

	shutdown := []func() error{
		a.discovery.Leave,
		func() error {
			a.server.GracefulStop()
			return nil
		},
	}
	for _, fn := range shutdown {
		if err := fn(); err != nil {
			a.logger.Printf("Error during shutdown: %v", err)
			return err
		}
	}
	a.logger.Println("Shutting down...")
	return nil
}
