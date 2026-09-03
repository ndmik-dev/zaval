package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ndmik-dev/zaval/internal/store"
	"github.com/ndmik-dev/zaval/internal/web"
)

func main() {
	loadDotEnv(envOr("ENV_FILE", ".env"))
	addr := envOr("ADDR", ":8080")

	// The container image has no shell or curl, so the binary checks itself.
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		if err := healthcheck(addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	password := os.Getenv("PASSWORD")
	if password == "" && !loopback(addr) && os.Getenv("INSECURE") == "" {
		log.Fatal("PASSWORD is empty on a non-loopback address; set PASSWORD (or INSECURE=1 to allow open access)")
	}

	st, err := store.Open(envOr("DB_PATH", "dayboard.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	if err := st.Seed(); err != nil {
		log.Fatal(err)
	}
	if dir := os.Getenv("BACKUP_DIR"); dir != "" {
		go nightlyBackups(st, dir)
	}

	srv := web.New(st, password)
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

// loadDotEnv reads KEY=VALUE lines into the environment; real environment
// variables always win over the file. A missing file is fine.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func healthcheck(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("healthz: " + res.Status)
	}
	return nil
}

// nightlyBackups writes a consistent copy of the database every night at 03:00
// local time and keeps the last 30.
func nightlyBackups(st *store.Store, dir string) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		time.Sleep(time.Until(next))
		if err := st.Backup(dir, 30); err != nil {
			log.Println("backup:", err)
		} else {
			log.Println("backup written to", dir)
		}
	}
}
