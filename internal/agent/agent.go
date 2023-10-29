package agent

import (
	"fmt"
	api "github.com/khatibomar/dhangkanna/api/v1"
	"github.com/khatibomar/dhangkanna/internal/discovery"
	"github.com/khatibomar/dhangkanna/internal/server"
	"github.com/khatibomar/dhangkanna/internal/state"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"os"
	"sync"
)

type Agent struct {
	Config

	State        *state.State
	server       *grpc.Server
	discovery    *discovery.Discovery
	replicator   *state.Replicator
	logger       *log.Logger
	shutdown     bool
	shutdowns    chan struct{}
	shutdownLock sync.Mutex
}

type Config struct {
	BindAddr       string
	RPCPort        int
	NodeName       string
	StartJoinAddrs []string
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
		State:     state.New(),
		shutdowns: make(chan struct{}),
		logger:    log.New(os.Stdout, "agent: ", log.LstdFlags),
	}
	setup := []func() error{
		a.setupServer,
		a.setupDiscovery,
	}
	for _, fn := range setup {
		if err := fn(); err != nil {
			return nil, err
		}
	}
	return a, nil
}

func (a *Agent) setupDiscovery() error {
	rpcAddr, err := a.Config.RPCAddr()
	if err != nil {
		a.logger.Printf("Error getting RPC address: %v", err)
		return err
	}
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(rpcAddr, opts...)
	if err != nil {
		a.logger.Printf("Error dialing RPC: %v", err)
		return err
	}
	client := api.NewStateServiceClient(conn)
	a.replicator = &state.Replicator{
		DialOptions: opts,
		LocalServer: client,
	}
	a.discovery, err = discovery.New(a.replicator, discovery.Config{
		NodeName: a.Config.NodeName,
		BindAddr: a.Config.BindAddr,
		Tags: map[string]string{
			"rpc_addr": rpcAddr,
		},
		StartJoinsAddresses: a.Config.StartJoinAddrs,
	})
	if err != nil {
		a.logger.Printf("Error creating Discovery: %v", err)
	}
	return err
}

func (a *Agent) setupServer() error {
	serverConfig := &server.Config{
		State: a.State,
	}
	var opts []grpc.ServerOption
	var err error
	a.server, err = server.NewGRPCServer(serverConfig, opts...)
	if err != nil {
		a.logger.Printf("Error creating gRPC server: %v", err)
		return err
	}
	rpcAddr, err := a.RPCAddr()
	if err != nil {
		a.logger.Printf("Error getting RPC address: %v", err)
		return err
	}
	ln, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		a.logger.Printf("Error listening on address: %v", err)
		return err
	}
	go func() {
		if err := a.server.Serve(ln); err != nil {
			a.logger.Printf("Error serving gRPC server: %v", err)
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
		a.replicator.Close,
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
	return nil
}
