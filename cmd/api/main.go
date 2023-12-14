package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/khatibomar/dhangkanna/internal/agent"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
)

var dataDir = path.Join(os.TempDir(), "dhangkanna")

func main() {
	logger := log.New(os.Stdout, "server: ", log.LstdFlags|log.Lshortfile)
	if err := run(); err != nil {
		logger.Fatal(err)
	}
}

func run() error {
	cfg := agent.Config{}
	parse(&cfg)

	_ = os.Mkdir(dataDir, 0666)

	addr, err := getRpcAddress(cfg)
	if err != nil {
		return err
	}

	if err := storeServerAddress(addr); err != nil {
		return err
	}

	defer func(addr string) {
		err := removeServerFromDB(addr)
		log.Printf("Removing %s from bucket\n", addr)
		if err != nil {
			log.Println(err)
		}
	}(addr)

	a, err := agent.New(cfg)
	if err != nil {
		return err
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			j, err := json.Marshal(a.DistributedGame.Game)
			if err != nil {
				log.Fatal(err)
			}
			_, _ = io.WriteString(w, string(j))
		})

		_ = http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", a.Config.RPCPort+1), nil)
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	return a.Shutdown()
}

func parse(cfg *agent.Config) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	flag.StringVar(&cfg.DataDir, "data-dir",
		"",
		"Directory to store log and Raft data.")
	flag.StringVar(&cfg.NodeName, "node-name", hostname, "Unique server ID.")
	flag.StringVar(&cfg.BindAddr, "bind-addr",
		"127.0.0.1:4001",
		"Address to bind Serf on.")
	flag.IntVar(&cfg.RPCPort, "rpc-port",
		4002,
		"Port for RPC clients (and Raft) connections.")
	var startAddrs string
	flag.StringVar(&startAddrs, "start-join-addrs",
		"",
		"Serf addresses to join.")

	flag.BoolVar(&cfg.Bootstrap, "bootstrap", false, "Bootstrap the cluster.")

	flag.Parse()

	if startAddrs != "" {
		cfg.StartJoinAddrs = strings.Split(startAddrs, ",")
	}
}

func removeServerFromDB(addr string) error {
	db, err := bolt.Open(path.Join(dataDir, "serverlist.db"), 0600, nil)
	if err != nil {
		return err
	}
	defer func(db *bolt.DB) {
		_ = db.Close()
	}(db)

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("ServerAddresses"))
		if bucket == nil {
			return fmt.Errorf("Bucket not found")
		}

		if err := bucket.Delete([]byte(addr)); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func storeServerAddress(addr string) error {
	db, err := bolt.Open(path.Join(dataDir, "serverlist.db"), 0600, nil)
	if err != nil {
		return err
	}
	defer func(db *bolt.DB) {
		err := db.Close()
		if err != nil {
			log.Println(err)
		}
	}(db)

	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("ServerAddresses"))
		if err != nil {
			return err
		}
		id, _ := bucket.NextSequence()
		key := itob(id)
		return bucket.Put(key, []byte(addr))
	})
}

func getRpcAddress(cfg agent.Config) (string, error) {
	host, _, err := net.SplitHostPort(cfg.BindAddr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", host, cfg.RPCPort), nil
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}
