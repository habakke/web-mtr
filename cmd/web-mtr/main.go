package main

import (
	"context"
	"encoding/json"
	"github.com/habakke/web-mtr/pkg/mtr"
	"github.com/habakke/web-mtr/pkg/semaphore"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var (
	COUNT            = 60
	RING_BUFFER_SIZE = 60
	TIMEOUT          = 800 * time.Millisecond
	INTERVAL         = 100 * time.Millisecond
	HOP_SLEEP        = time.Nanosecond
	MAX_HOPS         = 64
	MAX_UNKNOWN_HOPS = 10
	PTR_LOOKUP       = false
	srcAddr          = ""

	sem *semaphore.Semaphore

	listen      = getEnv("LISTEN_ADDRESS", ":8080")
	dir         = getEnv("WEB_DIR", "/opt/web")
	requireRoot = getEnv("REQUIRE_ROOT", "false")
)

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func traceHandler(w http.ResponseWriter, r *http.Request) {
	sem.Lock()            // acquire a resource
	defer sem.Unlock()    // release a resource when we finish

	ip, ok := r.URL.Query()["ip"]
	if !ok || len(ip[0]) < 1 {
		log.Println("url param 'ip' is missing")
		w.WriteHeader(400)
		return
	}

	log.Printf("serving dynamic URL '%s' to %s", r.URL.Path, r.RemoteAddr)
	m, ch, err := mtr.NewMTR(ip[0], srcAddr, TIMEOUT, INTERVAL, HOP_SLEEP, MAX_HOPS, MAX_UNKNOWN_HOPS, RING_BUFFER_SIZE, PTR_LOOKUP)
	defer close(ch)
	if err != nil {
		log.Println(err)
		w.WriteHeader(500)
		return
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, COUNT)

	s, _ := json.MarshalIndent(m.Statistic, "", "    ")
	_, err = w.Write(s)
}

func serve(ctx context.Context) (err error) {

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(dir)))
	mux.Handle("/trace", http.HandlerFunc(traceHandler))

	srv := &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen:%+s\n", err)
		}
	}()

	log.Println("server started")
	<-ctx.Done()
	log.Println("server stopped")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("server Shutdown Failed:%+s", err)
	}

	log.Printf("server exited properly")

	if err == http.ErrServerClosed {
		err = nil
	}

	return
}

func main() {
	if os.Getuid() != 0 && requireRoot == "true" {
		log.Fatalf("This program has to run as root (UID=%d)", os.Getuid())
	}

	sem = semaphore.NewSemaphore(3) // Semaphore to block requests if more than 3 traces are currently running
	defer sem.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-c
		cancel()
	}()

	if err := serve(ctx); err != nil {
		log.Printf("failed to serve:+%v\n", err)
	}
}
