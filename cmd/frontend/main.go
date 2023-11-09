package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type serverConfig struct {
	port int
}

func main() {
	cfg := &serverConfig{}
	flag.IntVar(&cfg.port, "port", 4000, "port that socket will run on")
	flag.Parse()

	serverLogger := log.New(os.Stdout, "frontend: ", log.LstdFlags|log.Lshortfile)
	if err := serve(cfg, serverLogger); err != nil {
		serverLogger.Fatalf("%v", err)
	}
}

func serve(cfg *serverConfig, serverLogger *log.Logger) error {
	fs := http.FileServer(http.Dir("static"))

	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "game.html")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n, err := NewSocket(ctx)
	if err != nil {
		return err
	}

	http.HandleFunc("/ws", n.HandleWebSocket)

	address := fmt.Sprintf(":%d", cfg.port)

	serverLogger.Printf("Server is running on port %d\n", cfg.port)
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			if err != nil {
				serverLogger.Println(err)
			}
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	return nil
}
