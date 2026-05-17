package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadingRelay(t *testing.T) {
	payload := make(chan ReadingPayload)

	// #region Initialize Test Callback Server
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		values := r.Form
		payload <- ReadingPayload{
			timestamp:    values.Get("timestamp"),
			temperature:  values.Get("temperature"),
			humidity:     values.Get("humidity"),
			air_pressure: values.Get("air_pressure"),
			brightness:   values.Get("brightness"),
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer cbServer.Close()
	// #endregion

	// #region Initialize Test Server
	// Use Mux to extract path param
	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(cbServer.URL, []byte("TEST_SECRET")))
	// #endregion

	testCases := []struct {
		name     string
		payload  ReadingPayload
		expected int
	}{
		{
			name: "Happy Path",
			payload: ReadingPayload{
				timestamp:    strconv.FormatInt(time.Now().Unix(), 10),
				humidity:     "65.25",
				temperature:  "27.01",
				air_pressure: "15.94",
				brightness:   "5380",
			},
			expected: 202,
		},
		{
			name: "Leading Decimal",
			payload: ReadingPayload{
				timestamp:    strconv.FormatInt(time.Now().Unix(), 10),
				humidity:     ".65",
				temperature:  "27.01",
				air_pressure: "15.94",
				brightness:   "5380",
			},
			expected: 202,
		},
		{
			name: "Trailing Decimal",
			payload: ReadingPayload{
				timestamp:    strconv.FormatInt(time.Now().Unix(), 10),
				humidity:     "65.",
				temperature:  "27.01",
				air_pressure: "15.94",
				brightness:   "5380",
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
				timestamp:    strconv.FormatInt(time.Now().Unix(), 10),
				humidity:     "wet",
				temperature:  "27.01",
				air_pressure: "15.94",
				brightness:   "5380",
			},
			expected: 400,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			body := strings.NewReader(url.Values{
				"timestamp":    {testCase.payload.timestamp},
				"humidity":     {testCase.payload.humidity},
				"temperature":  {testCase.payload.temperature},
				"air_pressure": {testCase.payload.air_pressure},
				"brightness":   {testCase.payload.brightness},
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
			if relayed.timestamp != testCase.payload.timestamp {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.humidity != testCase.payload.humidity {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.temperature != testCase.payload.temperature {
				t.Errorf("Unexpected temperature value")
			}
			if relayed.air_pressure != testCase.payload.air_pressure {
				t.Errorf("Unexpected air_pressure value")
			}
			if relayed.brightness != testCase.payload.brightness {
				t.Errorf("Unexpected brightness value")
			}
			// #endregion
		})

	}

}
