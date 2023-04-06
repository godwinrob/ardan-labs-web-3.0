package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"go.uber.org/automaxprocs/maxprocs"
)

var build = "develop"

func main() {
	if _, err := maxprocs.Set(); err != nil {
		log.Println("failed to set maxprocs")
		os.Exit(1)
	}

	cores := runtime.GOMAXPROCS(0)
	log.Printf("service started. build=%s, CPUs=%d", build, cores)
	defer log.Print("service stopped")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Println("stopping service")
}
