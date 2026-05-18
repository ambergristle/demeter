package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	env, err := initEnv()
	if err != nil {
		log.Fatalln(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(env.callbackUrl))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", env.port),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		// IdleTimeout:  120 * time.Second,
		Handler: http.MaxBytesHandler(mux, 500),
	}

	log.Printf("Server listening on %s\n", srv.Addr)
	log.Fatalln(srv.ListenAndServe())
}

func initEnv() (SafeEnv, error) {
	var callbackUrl string
	flag.StringVar(&callbackUrl, "cb", "", "The callback URL readings are relayed to.")

	var port int
	flag.IntVar(&port, "p", 8080, "The port to serve backend on, default: `8080`.")

	var secret string
	flag.StringVar(&secret, "s", "", "The secret used to sign callback requests.")

	flag.Parse()

	if callbackUrl == "" {
		return SafeEnv{}, errors.New("Missing required flag: -cb")
	}

	if secret == "" {
		return SafeEnv{}, errors.New("Missing required flag: -s")
	}

	os.Setenv("SIGNING_SECRET", secret)

	return SafeEnv{callbackUrl: callbackUrl, port: port}, nil
}

type SafeEnv struct {
	callbackUrl string
	port        int
}
