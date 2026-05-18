package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	var callbackUrl string
	flag.StringVar(&callbackUrl, "cb", "", "The callback URL readings are relayed to.")

	var secret string
	flag.StringVar(&secret, "s", "", "The secret used to sign callback requests.")

	var port int
	flag.IntVar(&port, "p", 8080, "The port to serve backend on, default: `8080`.")

	flag.Parse()

	if len(callbackUrl) == 0 {
		log.Fatalf("missing required flag: -cb")
	}

	if len(secret) == 0 {
		log.Fatalf("missing required flag: -s")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(callbackUrl, []byte(secret)))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		// IdleTimeout:  120 * time.Second,
		Handler: http.MaxBytesHandler(mux, 500),
	}

	log.Printf("Server listening on %s\n", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
