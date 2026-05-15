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
	"regexp"
	"strconv"
	"strings"
	"time"
)

func server(options *DemeterOptions) {
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
			fmt.Printf("client id: %s\n", sensorId)

			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			// Using `.Form` prioritizes body data
			// over URL params
			reading, err := parseReading(&r.Form)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			err = dispatchEvent(callbackUrl, reading, []byte(secret))
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func parseReading(values *url.Values) (ReadingPayload, error) {
	timestamp := values.Get("timestamp")
	if !matchNumeric(timestamp) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: timestamp")
	}

	humidity := values.Get("humidity")
	if !matchNumeric(humidity) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: humidity")
	}

	temperature := values.Get("temperature")
	if !matchNumeric(humidity) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: humidity")
	}

	air_pressure := values.Get("air_pressure")
	if !matchNumeric(humidity) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: humidity")
	}

	brightness := values.Get("brightness")
	if !matchNumeric(humidity) {
		return ReadingPayload{}, fmt.Errorf("Invalid parameter: humidity")
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
	re := regexp.MustCompile(`^\d+(\.\d+){0,1}$`)
	match := re.MatchString(text)
	return match
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
	req, err := http.NewRequest("POST", callbackUrl, strings.NewReader(bodyString))

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("content-length", strconv.Itoa(len(bodyString)))

	req.Header.Add("x-demeter-timestamp", timestamp)
	req.Header.Add("x-demeter-signature", signature)
	// #endregion

	res, err := client.Do(req)

	if err != nil {
		return errors.New("Event dispatch failed")
	}

	if res.StatusCode != 200 {
		return errors.New("Event dispatch failed")
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
