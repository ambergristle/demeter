package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestReadingRelay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	body := strings.NewReader(url.Values{
		"timestamp":    {"1778870328"},
		"humidity":     {"50.0"},
		"temperature":  {"30.35"},
		"air_pressure": {"0"},
		"brightness":   {"10"},
	}.Encode())

	req := httptest.NewRequest("POST", "http://localhost:8080/readings/TEST_ID", body)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	readingHandler(server.URL, "TEST_SECRET")(w, req)

	resp := w.Result()

	if resp.StatusCode != 200 {
		t.Errorf("Callback failed with status: %d", resp.StatusCode)
	}
}
