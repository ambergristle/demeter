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
	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer cbServer.Close()

	// #region Construct Request
	body := strings.NewReader(url.Values{
		"timestamp":    {strconv.FormatInt(time.Now().Unix(), 10)},
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
}
