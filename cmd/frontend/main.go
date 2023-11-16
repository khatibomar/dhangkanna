package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

//go:embed static
var staticFolder embed.FS

//go:embed game.html
var gameHTML embed.FS

type serverConfig struct {
	port        int
	backendAddr []string
}

func main() {
	cfg := &serverConfig{}
	flag.IntVar(&cfg.port, "port", 4000, "port that socket will run on")
	var addrs string
	flag.StringVar(&addrs, "backend-addr", "", "backend addresses are comma seperated, use in case you don't need to auto pick one")
	flag.Parse()

	if addrs != "" {
		cfg.backendAddr = strings.Split(addrs, ",")
	}

	serverLogger := log.New(os.Stdout, "frontend: ", log.LstdFlags|log.Lshortfile)
	if err := serve(cfg, serverLogger); err != nil {
		serverLogger.Fatalf("%v", err)
	}
}

func serve(cfg *serverConfig, serverLogger *log.Logger) error {
	fs := http.FileServer(http.FS(staticFolder))

	http.Handle("/static/", http.StripPrefix("/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, err := gameHTML.ReadFile("game.html")
		if err != nil {
			http.Error(w, "Could not read game.html", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(content)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n, err := NewSocket(ctx, cfg.backendAddr)
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
