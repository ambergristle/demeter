package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
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

// Include unique ID in client call? Or assign here?
type ReadingPayload struct {
	Timestamp   string `json:"timestamp"`
	Humidity    string `json:"humidity"`
	Temperature string `json:"temperature"`
	AirPressure string `json:"air_pressure"`
	Brightness  string `json:"brightness"`
}

// #region Validation

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

// #endregion

func postCallback(
	client *http.Client,
	cbUrl string,
	payload ReadingPayload,
) error {
	var (
		// Unique ID for each reading event
		eid  string = generateUniqueID()
		err  error
		resp *http.Response
	)

	// Create json first to generate signature
	bodyBytes, err := json.Marshal(
		Payload{
			Type:      "reading",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Data:      payload,
		},
	)
	if err != nil {
		return err
	}
	// Create once for any/all attempts
	bodyBuff := bytes.NewBuffer(bodyBytes)
	bodyStr := string(bodyBytes)

	// Allow up to 3 retries, for >= 500
	err = withRetry(3, time.Sleep, func() (bool, error) {
		// #region Construct Request
		var req *http.Request
		req, err = http.NewRequest("POST", cbUrl, bodyBuff)
		if err != nil {
			// Client or configuration error,
			// Exit early
			return false, err
		}

		// Unique timestamp, signature for each attempt
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		sig := generateSignature(eid, ts, bodyStr)

		req.Header.Add("content-type", "application/json")
		// Standard Webhooks signature headers
		req.Header.Add("webhook-id", eid)
		req.Header.Add("webhook-timestamp", ts)
		req.Header.Add("webhook-signature", sig)
		// #endregion

		resp, err = client.Do(req)
		if err != nil {
			// Client or configuration error
			return false, err
		}
		defer resp.Body.Close()
		// Fail if body has length?

		if resp.StatusCode >= 500 {
			return true, nil
		}
		return false, nil // Success or non-recoverable error
	})

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if resp.StatusCode == http.StatusGone {
		// Unsubscribe
	}

	return errors.New("Callback failed with status: " + resp.Status)
}

type Payload struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"` // ISO 8601
	Data      ReadingPayload `json:"data"`
}

// generateSignature creates an HMAC signature and timestamp.
//
// eid is an event ID unique to each reading relay.
//
// ts is the Unix timestamp for the POST attempt.
//
// body is idiomatically JSON string, but could be
// form-url-encoded as long as client and server agree.
func generateSignature(eid string, ts string, body string) string {
	sigBase := fmt.Sprintf("%s.%s.%s", eid, ts, body)

	secret := []byte(os.Getenv("SIGNING_SECRET"))
	// This only clears the bytes, not heap string for `Getenv`.
	// Strings in go are immutable, can't be cleared.
	defer func() { clear(secret) }()

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(sigBase))

	return fmt.Sprintf("v1,%s", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
}

func generateUniqueID() string {
	// Use length divisible by 3
	// to avoid padding
	buff := make([]byte, 18)
	ts := time.Now().Unix()
	// Put the timestamp in the first 4 bytes
	binary.LittleEndian.PutUint32(buff, uint32(ts))
	// Read random data into the rest
	rand.Read(buff[3:])

	return base64.StdEncoding.EncodeToString(buff)
}

func generateSecret() string {
	b := make([]byte, 64)
	rand.Read(b)
	return "whsec_" + base64.StdEncoding.EncodeToString(b)
}

func withRetry(maxTries int, sleep func(time.Duration), fn func() (bool, error)) error {
	for tries := 0; tries < maxTries; tries++ {
		retry, err := fn()
		if err != nil {
			return err
		}
		if !retry {
			return nil
		}
		sleep(time.Second * time.Duration(math.Pow(2, float64(tries))))
	}
	return errors.New("Max retries exceeded.")
}
