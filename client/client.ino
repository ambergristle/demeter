#include <ArduinoHttpClient.h>
#include <Arduino_MKRIoTCarrier.h>
#include <WiFiNINA.h>

#include <SPI.h>
#include <utility/wifi_drv.h>

#include <ctime>

#include "client.h"
#include "secrets.h"

MKRIoTCarrier carrier;

WiFiSSLClient wifi;
int wifiStatus = WL_IDLE_STATUS;

HttpClient client = HttpClient(wifi, SERVICE_HOST, SERVICE_PORT);

void setup() {
    // #region Connect to board
    Serial.begin(9600); // Specify output port (default)
    while (!Serial);

    Serial.println("\nInitializing...");
    carrier.noCase();
    carrier.begin();
    // #endregion

    // #region Connect to WiFi
    Serial.println("Attempting to connect to WiFI...");
    if (WiFi.status() == WL_NO_MODULE) {
        Serial.println("WiFi module unavailable.");
        while (true);
    }

    if (WiFi.firmwareVersion() < WIFI_FIRMWARE_LATEST_VERSION) {
        Serial.println("WiFi firmware must be updated.");
    }

    while (wifiStatus != WL_CONNECTED) {
        wifiStatus = WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
        delay(1000 * 5);
    }
    Serial.println("Connected");
    // #endregion
}

int postReading(HttpClient client, Reading reading) {
    String body = "timestamp=" + reading.timestamp;
    body += "&temperature=" + String(reading.temperature);
    body += "&humidity=" + String(reading.humidity);
    body += "&air_pressure=" + String(reading.air_pressure);
    body += "&brightness=" + String(reading.brightness);

    client.beginRequest();
    client.post("/readings/" + SENSOR_ID);
    client.sendHeader("content-type", "application/x-www-form-urlencoded");
    client.sendHeader("content-length", body.length());
    client.beginBody();
    client.print(body);
    client.endRequest();

    return client.responseStatusCode();
}

Reading takeReading() {
    Reading reading = Reading();

    // https://docs.arduino.cc/libraries/arduino_apds9960/
    while (!carrier.Light.colorAvailable()) {
        // todo: Add timeout
        delay(5);
    }

    time_t now;
    time(&now);
    reading.timestamp = ctime(&now);

    carrier.Light.readColor(
        reading.light_color.r,
        reading.light_color.g,
        reading.light_color.b, 
        reading.brightness
    );

    // https://docs.arduino.cc/libraries/arduino_hts221/
    reading.temperature = carrier.Env.readTemperature();
    reading.humidity = carrier.Env.readHumidity();
    // https://docs.arduino.cc/libraries/arduino_lps22hb/
    reading.air_pressure = carrier.Pressure.readPressure();

    return reading;
}

void loop() {
    Reading reading = takeReading();
    int status = postReading(client, reading);
    Serial.println("Response Status: ");
    delay(1000 * 5);
}
