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
)

type serverConfig struct {
	port uint
}

func main() {
	cfg := serverConfig{}

	flag.UintVar(&cfg.port, "port", 4000, "Specify the port on which the game will run")

	flag.Parse()

	serverLogger := log.New(os.Stdout, "Server: ", log.LstdFlags)

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
	var sja []string
	if cfg.port != 4000 {
		sja = []string{"127.0.0.1:4001"}
	}
	aCfg := agent.Config{
		BindAddr:       fmt.Sprintf("127.0.0.1:%d", int(cfg.port+1)),
		RPCPort:        int(cfg.port + 2),
		NodeName:       fmt.Sprintf("node%d", &cfg.port),
		StartJoinAddrs: sja,
	}
	s, err := node.New(ctx, aCfg)
	if err != nil {
		return err
	}
	http.HandleFunc("/ws", s.HandleWebSocket)

	address := fmt.Sprintf(":%d", cfg.port)

	serverLogger.Printf("Server is running on port %d\n", cfg.port)
	return http.ListenAndServe(address, nil)
}
