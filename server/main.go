package main

import "flag"

func main() {
	var callbackUrl string
	flag.StringVar(&callbackUrl, "cb", "", "The callback URL readings are relayed to.")
	if len(callbackUrl) == 0 {
		panic("No callback URL configured")
	}

	var port int
	flag.IntVar(&port, "p", 8080, "The port to serve backend on, default: `8080`.")

	var secret string
	flag.StringVar(&secret, "p", "", "The secret used to sign callback requests.")
	if len(secret) == 0 {
		panic("No signing secret configured")
	}

	flag.Parse()

	server(&DemeterOptions{
		callbackUrl: callbackUrl,
		port:        port,
		secret:      secret,
	})
}
