package main

import "flag"

func main() {
	var callbackUrl string
	flag.StringVar(&callbackUrl, "cb", "", "The callback URL readings are relayed to")
	flag.Parse()

	// Validate configured callback URL
	if len(callbackUrl) == 0 {
		panic("No callback URL configured")
	}

	server(&DemeterOptions{
		callbackUrl: "http://localhost:8000",
		port:        8080,
		secret:      "MOCK_SECRET",
	})
}
