package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/khatibomar/dhangkanna/internal/agent"
	"github.com/khatibomar/dhangkanna/internal/node"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

type serverConfig struct {
	port        uint
	agentConfig agent.Config
}

func main() {
	cfg := serverConfig{}
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	var startJoinAddrs string
	dataDir := path.Join(os.TempDir(), "dhangkanna")

	flag.UintVar(&cfg.port, "port", 4000, "Specify the port on which the game will run")
	flag.StringVar(&cfg.agentConfig.NodeName, "node-name", hostname, "Unique server ID.")
	flag.StringVar(&cfg.agentConfig.DataDir, "data-dir", dataDir, "Directory to store Raft data.")
	flag.StringVar(&cfg.agentConfig.BindAddr, "bind-addr", "127.0.0.1:4001", "Address to bind Serf on.")
	flag.IntVar(&cfg.agentConfig.RPCPort, "rpc-port", 4002, "Port for RPC clients (and Raft) connections.")
	flag.BoolVar(&cfg.agentConfig.Bootstrap, "bootstrap", false, "Bootstrap the cluster.")
	flag.StringVar(&startJoinAddrs, "start-join-addrs", "", "Serf addresses to join.")
	flag.Parse()

	if startJoinAddrs != "" {
		cfg.agentConfig.StartJoinAddrs = strings.Split(startJoinAddrs, ",")
	}

	serverLogger := log.New(os.Stdout, "server: ", log.LstdFlags)

	if err := serve(cfg, serverLogger); err != nil {
		serverLogger.Fatalf("%v", err)
	}

}

func serve(cfg serverConfig, serverLogger *log.Logger) error {
	fs := http.FileServer(http.Dir("static"))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "game.html")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n, err := node.New(ctx, cfg.agentConfig)
	if err != nil {
		return err
	}

	http.HandleFunc("/ws", n.HandleWebSocket)

	address := fmt.Sprintf(":%d", cfg.port)

	serverLogger.Printf("Server is running on port %d\n", cfg.port)
	go func() {
		if err = http.ListenAndServe(address, nil); err != nil {
			err := n.Shutdown()
			if err != nil {
				serverLogger.Println(err)
			}
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	return n.Shutdown()
}
