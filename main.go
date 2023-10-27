package main

import (
	"context"
	"flag"
	"fmt"
	state "github.com/khatibomar/dhangkanna/internal/node"
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

	s := state.New(ctx)
	http.HandleFunc("/ws", s.HandleWebSocket)

	address := fmt.Sprintf(":%d", cfg.port)

	serverLogger.Printf("Server is running on port %d\n", cfg.port)
	return http.ListenAndServe(address, nil)
}
