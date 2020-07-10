package main

import (
	"context"
	"github.com/habakke/web-mtr/pkg/webMTR"
	"log"
	"os"
	"os/signal"
)

var (
	REQUEST_COUNT int = 60

	wm *webMTR.WebMTR

	listenPort  = getEnv("LISTEN_ADDRESS", ":8080")
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

func isRoot() bool {
	return os.Getuid() == 0
}

func main() {
	if !isRoot() && requireRoot == "true" {
		log.Fatalf("This program has to run as root (current UID=%d)", os.Getuid())
	}

	wm = webMTR.NewWebMTR(listenPort, dir, REQUEST_COUNT)
	defer wm.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-c
		cancel()
	}()

	if err := wm.Serve(ctx); err != nil {
		log.Printf("failed to serve:+%v\n", err)
	}
}
