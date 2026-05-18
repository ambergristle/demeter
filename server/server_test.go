package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadingRelay(t *testing.T) {
	os.Setenv("SIGNING_SECRET", "TEST_SECRET")
	// Pass through channel to guarantee availability
	payload := make(chan ReadingPayload)

	// #region Initialize Test Callback Server
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)

		var reading ReadingPayload
		err := decoder.Decode(&reading)
		if err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		io.Copy(io.Discard, r.Body)
		r.Body.Close()

		payload <- reading
		log.Printf("reading: %+v\n", reading)

		w.WriteHeader(http.StatusOK)
	}))
	defer cbServer.Close()
	// #endregion

	// #region Initialize Test Server
	// Use Mux to extract path param
	client := &http.Client{
		Timeout: time.Second * 3,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(client, cbServer.URL))
	// #endregion

	testCases := []struct {
		name     string
		payload  ReadingPayload
		expected int
	}{
		{
			name: "Happy Path",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.25",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 202,
		},
		{
			name: "Leading Decimal",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    ".65",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 202,
		},
		{
			name: "Trailing Decimal",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 202,
		},
		{
			name:     "Bad Request",
			payload:  ReadingPayload{},
			expected: 400,
		},
		{
			name: "Not A Number",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "wet",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 400,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body := strings.NewReader(url.Values{
				"timestamp":    {testCase.payload.Timestamp},
				"humidity":     {testCase.payload.Humidity},
				"temperature":  {testCase.payload.Temperature},
				"air_pressure": {testCase.payload.AirPressure},
				"brightness":   {testCase.payload.Brightness},
			}.Encode())

			req := httptest.NewRequest("POST", "/readings/TEST_ID", body)
			req.Header.Add("content-type", "application/x-www-form-urlencoded")

			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != testCase.expected {
				t.Errorf("Expected status %d, received %d", testCase.expected, resp.StatusCode)
			}

			if resp.StatusCode != 202 {
				return
			}

			// #region Ensure payload is relayed faithfully
			relayed := <-payload
			log.Printf("relayed: %+v\n", relayed)
			if relayed.Timestamp != testCase.payload.Timestamp {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.Humidity != testCase.payload.Humidity {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.Temperature != testCase.payload.Temperature {
				t.Errorf("Unexpected temperature value")
			}
			if relayed.AirPressure != testCase.payload.AirPressure {
				t.Errorf("Unexpected air_pressure value")
			}
			if relayed.Brightness != testCase.payload.Brightness {
				t.Errorf("Unexpected brightness value")
			}
			// #endregion
		})

	}

}

func TestRetry(t *testing.T) {
	os.Setenv("SIGNING_SECRET", "TEST_SECRET")
	// #region Initialize Test Callback Server
	errs := 0

	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if errs < 2 {
			errs += 1
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer cbServer.Close()
	// #endregion

	pay := ReadingPayload{
		Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
		Humidity:    "65.",
		Temperature: "27.01",
		AirPressure: "15.94",
		Brightness:  "5380",
	}

	client := &http.Client{
		Timeout: time.Second * 3,
	}
	err := postCallback(client, cbServer.URL, pay)
	if err != nil {
		t.Errorf("Unexpected error: %+v", err)
	}

	if errs != 2 {
		t.Errorf("Expected 3 retries")
	}
}
