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
	"time"

	"grade5math/internal/mathapp"
)

var revision = "dev"
var builtAt = "unknown"

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "listen address")
	contentDir := flag.String("content", "../content", "content directory")
	webDir := flag.String("web", "../frontend/dist", "built frontend")
	dbPath := flag.String("db", "../var/math.db", "SQLite file")
	check := flag.Bool("check", false, "validate content and exit")
	secureCookies := flag.Bool("secure-cookies", false, "require HTTPS for anonymous profile cookies")
	flag.Parse()
	content, err := mathapp.LoadContent(*contentDir)
	if err != nil {
		log.Fatal(err)
	}
	if *check {
		fmt.Printf("valid: %d modules, %d skills, %d questions\n", len(content.Modules), len(content.Skills), len(content.Questions))
		return
	}
	store, err := mathapp.OpenStore(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	app := mathapp.App{Content: content, Store: store, WebDir: *webDir, SecureCookies: *secureCookies, Revision: revision, BuiltAt: builtAt}
	server := &http.Server{Addr: *address, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	stop := make(chan os.Signal, 1)
	shutdownDone := make(chan struct{})
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		close(shutdownDone)
	}()
	log.Printf("小步数学 http://%s (content %s)", *address, content.Version)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-shutdownDone
}
