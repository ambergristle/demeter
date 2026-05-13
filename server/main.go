package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/readings/{sensorId}", func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case "POST":
			sensorId := r.PathValue("sensorId")
			fmt.Printf("client id: %s\n", sensorId)

			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			values := r.Form

			// #region Parse values
			timestamp := values.Get("timestamp")

			humidity, err := strconv.ParseFloat(values.Get("humidity"), 32)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			temperature, err := strconv.ParseFloat(values.Get("temperature"), 32)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			air_pressure, err := strconv.ParseFloat(values.Get("air_pressure"), 32)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			brightness, err := strconv.ParseFloat(values.Get("brightness"), 32)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			var soil_moisture float64
			moisture := values.Get("soil_moisture")
			if len(moisture) > 0 {
				soil_moisture, err = strconv.ParseFloat(moisture, 32)
				if err != nil {
					http.Error(w, "Bad request", http.StatusBadRequest)
					return
				}
			} else {
				// This feels smelly
				// But needed something to put in
				// The `else` block
				soil_moisture = -1
			}

			reading := Reading{
				timestamp:     timestamp,
				humidity:      float32(humidity),
				temperature:   float32(temperature),
				air_pressure:  float32(air_pressure),
				brightness:    float32(brightness),
				soil_moisture: float32(soil_moisture),
			}
			// #endregion

			fmt.Printf("form data: %+v\n", reading)

			w.WriteHeader(http.StatusOK)
		default:
			// Should never happen
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	port := 8080
	fmt.Printf("Server is running at http://localhost:%d\n", port)
	// Dynamic port just for science
	// But maybe should be configurable
	err := http.ListenAndServe(":"+strconv.Itoa(port), mux)
	log.Fatal(err)
}

type Reading struct {
	timestamp string
	// Data
	humidity      float32
	temperature   float32
	air_pressure  float32
	brightness    float32
	soil_moisture float32
}
