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
		log.Fatalf("Callback URL must be configured")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(options.callbackUrl, options.secret))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", options.port),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		// IdleTimeout:  120 * time.Second,
		Handler: http.MaxBytesHandler(mux, 500),
	}

	log.Printf("Server listening on %s\n", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

func readingHandler(callbackUrl string, secret []byte) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			sensorId := r.PathValue("sensorId")
			if sensorId == "" {
				http.Error(w, "Path must include sensorId", http.StatusBadRequest)
				return
			}

			if err := r.ParseForm(); err != nil {
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

			go func(r ReadingPayload) {
				if err := dispatchEvent(callbackUrl, r, secret); err != nil {
					log.Printf("Callback failed repeatedly: %v ", err)
				}
			}(reading)

			w.WriteHeader(http.StatusAccepted)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

func parseReading(values *url.Values) (ReadingPayload, error) {
	ts := values.Get("timestamp")
	if !validateTimestamp(ts) {
		return ReadingPayload{}, errors.New("Invalid parameter: timestamp")
	}

	hum := values.Get("humidity")
	if !isFloatStr(hum) {
		return ReadingPayload{}, errors.New("Invalid parameter: humidity")
	}

	temp := values.Get("temperature")
	if !isFloatStr(temp) {
		return ReadingPayload{}, errors.New("Invalid parameter: temperature")
	}

	air := values.Get("air_pressure")
	if !isFloatStr(air) {
		return ReadingPayload{}, errors.New("Invalid parameter: air_pressure")
	}

	bri := values.Get("brightness")
	if !isIntStr(bri) {
		return ReadingPayload{}, errors.New("Invalid parameter: brightness")
	}

	return ReadingPayload{
		timestamp:    ts,
		temperature:  temp,
		humidity:     hum,
		air_pressure: air,
		brightness:   bri,
	}, nil
}

func isFloatStr(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func isIntStr(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 32)
	return err == nil
}

func validateTimestamp(timestamp string) bool {
	tsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	return abs(now-tsInt) <= 5*60
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func dispatchEvent(callbackUrl string, payload ReadingPayload, secret []byte) error {
	sigTS := strconv.FormatInt(time.Now().Unix(), 10)

	// Initialize bodyVals first to generate signature
	bodyVals := url.Values{
		"timestamp":    {payload.timestamp},
		"humidity":     {payload.humidity},
		"temperature":  {payload.temperature},
		"air_pressure": {payload.air_pressure},
		"brightness":   {payload.brightness},
	}
	bodyStr := bodyVals.Encode()

	sig := formatSignature(bodyStr, sigTS, secret)

	// #region Construct Request
	client := &http.Client{
		Timeout: time.Second * 3,
	}

	req, err := http.NewRequest("POST", callbackUrl, strings.NewReader(bodyStr))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("x-dmtr-timestamp", sigTS)
	req.Header.Add("x-dmtr-signature", sig)
	// #endregion

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return errors.New("Callback failed with status: " + res.Status)
	}

	return nil
}

func formatSignature(body string, ts string, secret []byte) string {
	signatureBase := fmt.Sprintf("v0:%s:%s", ts, body)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signatureBase))

	return fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil)))
}

type DemeterOptions struct {
	callbackUrl string
	port        int
	secret      []byte
}

type ReadingPayload struct {
	timestamp    string
	humidity     string
	temperature  string
	air_pressure string
	brightness   string
}
