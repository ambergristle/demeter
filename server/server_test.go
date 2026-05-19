package main

import (
	"encoding/json"
	"io"
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
	payload := make(chan Payload)

	// #region Initialize Test Callback Server
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)

		var reading Payload
		err := decoder.Decode(&reading)
		if err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		io.Copy(io.Discard, r.Body)
		r.Body.Close()

		payload <- reading
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
			name:     "Bad Request",
			payload:  ReadingPayload{},
			expected: 400,
		},
		{
			name: "Bad Timestamp",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(10, 10),
				Humidity:    "65.25",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 400,
		},
		{
			name: "Bad Humidity",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "wet",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 400,
		},
		{
			name: "Bad Temperature",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.25",
				Temperature: "hot",
				AirPressure: "15.94",
				Brightness:  "5380",
			},
			expected: 400,
		},
		{
			name: "Bad Air Pressure",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.25",
				Temperature: "27.01",
				AirPressure: "crushing",
				Brightness:  "5380",
			},
			expected: 400,
		},
		{
			name: "Bad Brightness (Float)",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.25",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "5380.0",
			},
			expected: 400,
		},
		{
			name: "Bad Brightness (NaN)",
			payload: ReadingPayload{
				Timestamp:   strconv.FormatInt(time.Now().Unix(), 10),
				Humidity:    "65.25",
				Temperature: "27.01",
				AirPressure: "15.94",
				Brightness:  "blinding",
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
			if relayed.Data.Timestamp != testCase.payload.Timestamp {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.Data.Humidity != testCase.payload.Humidity {
				t.Errorf("Unexpected humidity value")
			}
			if relayed.Data.Temperature != testCase.payload.Temperature {
				t.Errorf("Unexpected temperature value")
			}
			if relayed.Data.AirPressure != testCase.payload.AirPressure {
				t.Errorf("Unexpected air_pressure value")
			}
			if relayed.Data.Brightness != testCase.payload.Brightness {
				t.Errorf("Unexpected brightness value")
			}
			// #endregion
		})

	}

}

func TestRetry(t *testing.T) {
	os.Setenv("SIGNING_SECRET", "TEST_SECRET")

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	testCases := []struct {
		name     string
		status   int
		expected int
	}{
		{
			name:     "Happy Path",
			status:   http.StatusOK,
			expected: 0,
		},
		{
			name:     "Abandon Unrecoverable",
			status:   http.StatusBadRequest,
			expected: 1,
		},
		{
			name:     "Attempt Retry",
			status:   http.StatusInternalServerError,
			expected: 2,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// #region Initialize Test Callback Server
			errs := 0

			cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if errs < 2 {
					if testCase.status >= 400 {
						errs += 1
					}
					w.WriteHeader(testCase.status)
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

			postCallback(client, cbServer.URL, pay)
			if errs != testCase.expected {
				t.Errorf("Expected %d retries", testCase.expected)
			}
		})
	}
}
