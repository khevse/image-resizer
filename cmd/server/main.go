package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/khevse/image-resizer/description"
	"github.com/khevse/image-resizer/service/images"
)

func main() {

	port := flag.String("port", "8000", "server port")

	log.Printf(
		"Starting the service...\ncommit: %s, build time: %s, release: %s",
		description.Commit, description.BuildDate, description.Version,
	)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: images.New().Mux(),
	}
	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	log.Print("The service is ready to listen and serve address ", srv.Addr)

	killSignal := <-interrupt
	switch killSignal {
	case os.Interrupt:
		log.Print("Got SIGINT...")
	case syscall.SIGTERM:
		log.Print("Got SIGTERM...")
	}

	log.Print("The service is shutting down...")
	srv.Shutdown(context.Background())
	log.Print("Done")
}
