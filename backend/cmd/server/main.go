package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/usip/backend/internal/api"
	"github.com/usip/backend/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "", "SQLite database path (default: $HOME/.usip/usip.db)")
	flag.Parse()

	if *dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("cannot determine home dir: %v", err)
		}
		dir := filepath.Join(home, ".usip")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("cannot create data dir: %v", err)
		}
		*dbPath = filepath.Join(dir, "usip.db")
	}

	st, err := store.NewSQLite(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	router := api.NewRouter(st)

	log.Printf("USIP backend listening on %s  db=%s", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
