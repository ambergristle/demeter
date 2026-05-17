package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func server(options *DemeterOptions) {
	if len(options.callbackUrl) == 0 {
		panic("Callback URL must be configured")
	}

	port := strconv.Itoa(options.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(options.callbackUrl, options.secret))

	fmt.Printf("Server is running at http://localhost:%s\n", port)
	err := http.ListenAndServe(":"+port, mux)

	log.Fatal(err)
}

func readingHandler(callbackUrl string, secret string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			sensorId := r.PathValue("sensorId")
			if len(sensorId) < 1 {
				http.Error(
					w,
					"Bad request: Path must include `sensorId`: `/readings/{sensorId}`",
					http.StatusBadRequest,
				)
				return
			}

			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Using `.Form` prioritizes body data
			// over URL params
			reading, err := parseReading(&r.Form)
			if err != nil {
				http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
				return
			}

			ch := make(chan error)
			go func(r ReadingPayload) {
				ch <- dispatchEvent(callbackUrl, r, []byte(secret))
			}(reading)

			w.WriteHeader(http.StatusAccepted)

			cbErr := <-ch
			if cbErr != nil {
				fmt.Println("Callback failed repeatedly: " + cbErr.Error())
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func parseReading(values *url.Values) (ReadingPayload, error) {
	timestamp := values.Get("timestamp")
	if !validateTimestamp(timestamp) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: timestamp")
	}

	humidity := values.Get("humidity")
	if !matchNumeric(humidity) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: humidity")
	}

	temperature := values.Get("temperature")
	if !matchNumeric(temperature) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: temperature")
	}

	air_pressure := values.Get("air_pressure")
	if !matchNumeric(air_pressure) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: air_pressure")
	}

	brightness := values.Get("brightness")
	if !matchNumeric(brightness) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: brightness")
	}

	return ReadingPayload{
		timestamp:    timestamp,
		temperature:  temperature,
		humidity:     humidity,
		air_pressure: air_pressure,
		brightness:   brightness,
	}, nil
}

func matchNumeric(text string) bool {
	for i, r := range text {
		if r < '0' || r > '9' {
			if r == '.' && i > 0 && i < len(text)-1 {
				// If there are more characters,
				// validate that they are digits
				continue
			}
			return false
		}
	}
	return text != ""
}

func validateTimestamp(timestamp string) bool {
	stamp, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	stampYear := time.Unix(stamp, 0).Year()
	thisYear := time.Now().Year()

	return stampYear >= thisYear-1 && stampYear <= thisYear+1
}

func dispatchEvent(callbackUrl string, payload ReadingPayload, secret []byte) error {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Initialize body first to generate signature
	body := url.Values{
		"timestamp":    {payload.timestamp},
		"humidity":     {payload.humidity},
		"temperature":  {payload.temperature},
		"air_pressure": {payload.air_pressure},
		"brightness":   {payload.brightness},
	}

	bodyString := body.Encode()
	signature := formatSignature(bodyString, timestamp, secret)

	// #region Construct Request
	client := &http.Client{}

	// todo: better reader?
	req, err := http.NewRequest("POST", callbackUrl, strings.NewReader(bodyString))

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("content-length", strconv.Itoa(len(bodyString)))

	req.Header.Add("x-dmtr-timestamp", timestamp)
	req.Header.Add("x-dmtr-signature", signature)
	// #endregion

	res, err := client.Do(req)

	if err != nil {
		return errors.New("Dispatch failed: " + err.Error())
	}

	if res.StatusCode != 200 {
		return errors.New("Dispatch failed with status: " + res.Status)
	}

	return nil
}

func formatSignature(body string, timestamp string, secret []byte) string {
	signatureBase := "v0:" + timestamp + ":" + body

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signatureBase))

	signature := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return signature
}

type DemeterOptions struct {
	callbackUrl string
	port        int
	secret      string
}

type ReadingPayload struct {
	timestamp string
	// Data
	humidity     string
	temperature  string
	air_pressure string
	brightness   string
}
