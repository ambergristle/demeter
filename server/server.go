package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func readingHandler(client *http.Client, callbackUrl string) func(w http.ResponseWriter, r *http.Request) {
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
				if err := postCallback(client, callbackUrl, r); err != nil {
					log.Printf("Callback failed repeatedly: %v", err)
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
		Timestamp:   ts,
		Temperature: temp,
		Humidity:    hum,
		AirPressure: air,
		Brightness:  bri,
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

func postCallback(
	client *http.Client,
	cbUrl string,
	payload ReadingPayload,
) error {
	var (
		err  error
		resp *http.Response
	)

	// Create json first to generate signature
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	sig := generateSignature(string(bodyBytes))

	// #region Construct Request
	tries := 0
	for tries < 3 {
		var req *http.Request
		req, err = http.NewRequest("POST", cbUrl, bytes.NewBuffer(bodyBytes))
		if err != nil {
			// Client or configuration error,
			// Exit early
			break
		}

		req.Header.Add("content-type", "application/json")
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
	Timestamp   string `json:"timestamp"`
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
	AirPressure string `json:"air_pressure"`
	Brightness  string `json:"brightness"`
}

// generateSignature creates an HMAC signature and timestamp.
//
// Body is expected to be a form-url-encoded string.
func generateSignature(body string) HmacSignature {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBase := fmt.Sprintf("v0:%s:%s", ts, body)

	secret := []byte(os.Getenv("SIGNING_SECRET"))
	// This only clears the bytes, not heap string for `Getenv`.
	// Strings in go are immutable, can't be cleared.
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
