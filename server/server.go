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
	"os"
	"strconv"
	"strings"
	"time"
)

func readingHandler(callbackUrl string) func(w http.ResponseWriter, r *http.Request) {
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
				if err := postCallback(callbackUrl, r); err != nil {
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
	if !isValidTS(ts) {
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

func isValidTS(ts string) bool {
	tsInt, err := strconv.ParseInt(ts, 10, 64)
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

func postCallback(cbUrl string, payload ReadingPayload) error {
	// Initialize bodyVals first to generate signature
	bodyVals := url.Values{
		"timestamp":    {payload.timestamp},
		"humidity":     {payload.humidity},
		"temperature":  {payload.temperature},
		"air_pressure": {payload.air_pressure},
		"brightness":   {payload.brightness},
	}
	bodyStr := bodyVals.Encode()

	sig := formatSignature(bodyStr)

	// #region Construct Request
	// Should this be happening for each request?
	client := &http.Client{
		Timeout: time.Second * 3,
		// Block redirects -- they're invalid
		// CheckRedirect: ,
	}

	var (
		req  *http.Request
		resp *http.Response
		err  error
	)
	tries := 0
	for tries < 3 {
		req, err = http.NewRequest("POST", cbUrl, strings.NewReader(bodyStr))
		if err != nil {
			// Client or configuration error,
			// Exit early
			break
		}

		req.Header.Add("content-type", "application/x-www-form-urlencoded")
		req.Header.Add("x-dmtr-timestamp", sig.timestamp)
		req.Header.Add("x-dmtr-signature", sig.signature)
		// #endregion

		resp, err = client.Do(req)
		if err != nil {
			// Client or configuration error,
			// Exit early
			break
		}
		defer resp.Body.Close()
		// Fail if body has length?

		if resp.StatusCode < 500 {
			// Success, or non-recoverable error,
			// Exit early
			break
		}

		backoff := time.Duration(1 * (2 ^ tries))
		time.Sleep(backoff)
		tries += 1
	}

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return errors.New("Callback failed with status: " + resp.Status)
}

type ReadingPayload struct {
	timestamp    string
	humidity     string
	temperature  string
	air_pressure string
	brightness   string
}

func formatSignature(body string) HmacSignature {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBase := fmt.Sprintf("v0:%s:%s", ts, body)

	secret := []byte(os.Getenv("SIGNING_SECRET"))
	// This only clears the bytes;
	// The string allocated for `Getenv` is inaccessible
	defer func() { clear(secret) }()

	mac := hmac.New(sha256.New, secret)

	mac.Write([]byte(sigBase))

	return HmacSignature{
		signature: fmt.Sprintf("v0=%s", hex.EncodeToString(mac.Sum(nil))),
		timestamp: ts,
	}
}

type HmacSignature struct {
	signature string
	timestamp string
}
