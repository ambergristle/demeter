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
	var payload ReadingPayload

	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		values := r.Form
		payload = ReadingPayload{
			timestamp:    values.Get("timestamp"),
			temperature:  values.Get("temperature"),
			humidity:     values.Get("humidity"),
			air_pressure: values.Get("air_pressure"),
			brightness:   values.Get("brightness"),
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer cbServer.Close()

	// #region Construct Request
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	body := strings.NewReader(url.Values{
		"timestamp":    {timestamp},
		"humidity":     {"50.0"},
		"temperature":  {"30.35"},
		"air_pressure": {"0"},
		"brightness":   {"10"},
	}.Encode())

	req := httptest.NewRequest("POST", "/readings/TEST_ID", body)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	// #endregion

	// Use Mux to extract path param
	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", readingHandler(cbServer.URL, "TEST_SECRET"))

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Callback failed with status: %d", resp.StatusCode)
	}

	// todo: hardcoding bad
	if payload.timestamp != timestamp {
		t.Errorf("Unexpected humidity value")
	}
	if payload.humidity != "50.0" {
		t.Errorf("Unexpected humidity value")
	}
	if payload.temperature != "30.35" {
		t.Errorf("Unexpected temperature value")
	}
	if payload.air_pressure != "0" {
		t.Errorf("Unexpected air_pressure value")
	}
	if payload.brightness != "10" {
		t.Errorf("Unexpected brightness value")
	}
}
