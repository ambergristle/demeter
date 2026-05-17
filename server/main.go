package main

import (
	"flag"
	"log"
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
		log.Fatalf("No callback URL configured")
	}

	if len(secret) == 0 {
		log.Fatalf("No signing secret configured")
	}

	server(&DemeterOptions{
		callbackUrl: callbackUrl,
		port:        port,
		secret:      []byte(secret),
	})
}
