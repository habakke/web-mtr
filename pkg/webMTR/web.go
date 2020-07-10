package webMTR

import (
	"context"
	"encoding/json"
	"github.com/habakke/web-mtr/pkg/mtr"
	"github.com/habakke/web-mtr/pkg/semaphore"
	"log"
	"net/http"
	"time"
)

var (
	NUM_CONCURRENT_REQS = 3
	TIMEOUT             = 800 * time.Millisecond
	INTERVAL            = 100 * time.Millisecond
	HOP_SLEEP           = time.Nanosecond
	MAX_HOPS            = 64
	MAX_UNKNOWN_HOPS    = 10
	PTR_LOOKUP          = false
	srcAddr             = ""
)

type WebMTR struct {
	sem        *semaphore.Semaphore
	count      int
	listenPort string
	dir        string
}

func NewWebMTR(listenPort string, dir string, requestCount int) *WebMTR {
	return &WebMTR{
		sem:        semaphore.NewSemaphore(NUM_CONCURRENT_REQS), // Semaphore to block requests if more than NUM_CONCURRENT_REQS traces are currently running
		listenPort: listenPort,
		dir:        dir,
		count:      requestCount,
	}
}

func (wm *WebMTR) Close() {
	wm.sem.Close()
}

func (wm *WebMTR) traceHandler(w http.ResponseWriter, r *http.Request) {
	wm.sem.Lock()         // acquire a resource
	defer wm.sem.Unlock() // release a resource when we finish

	ip, ok := r.URL.Query()["ip"]
	if !ok || len(ip[0]) < 1 {
		log.Println("url param 'ip' is missing")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(ip[0]) > 255 {
		log.Println("url param 'ip' is too long")
		w.WriteHeader(http.StatusBadRequest)
	}

	log.Printf("serving dynamic URL '%s' to %s", r.URL.Path, r.RemoteAddr)
	m, ch, err := mtr.NewMTR(ip[0], srcAddr, TIMEOUT, INTERVAL, HOP_SLEEP, MAX_HOPS, MAX_UNKNOWN_HOPS, wm.count, PTR_LOOKUP)
	defer close(ch)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func(ch chan struct{}) {
		for {
			<-ch
		}
	}(ch)
	m.Run(ch, wm.count)

	s, _ := json.MarshalIndent(m.Statistic, "", "    ")
	_, err = w.Write(s)
}

func (wm *WebMTR) Serve(ctx context.Context) (err error) {

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(wm.dir)))
	mux.Handle("/trace", http.HandlerFunc(wm.traceHandler))

	srv := &http.Server{
		Addr:    wm.listenPort,
		Handler: mux,
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listenPort:%+s\n", err)
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
