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

	"service-request-dispatch/internal/audit"
	"service-request-dispatch/internal/httpapi"
	"service-request-dispatch/internal/queue"
	"service-request-dispatch/internal/service"
	"service-request-dispatch/internal/store"
)

func main() {
	path := flag.String("db", "service-requests.db", "Bolt database path")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()
	db, err := store.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	auditor := audit.New(db)
	q := queue.New(32)
	app := service.New(db, auditor, q)
	handler := httpapi.New(app)
	server := &http.Server{Addr: *addr, Handler: handler, ReadHeaderTimeout: 3 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	}()
	if *addr == "" {
		fmt.Fprintln(os.Stdout, "service request dispatcher")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	q.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
