package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ndmik-dev/zaval/internal/web"
)

func main() {
	addr := envOr("ADDR", ":8080")
	srv := web.New()
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
