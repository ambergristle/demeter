#include <Arduino_MKRIoTCarrier.h>

void takeReading(*Reading reading) {
    // https://docs.arduino.cc/libraries/arduino_apds9960/
    while (!carrier.Light.colorAvailable) {
        // todo: Add timeout
        delay(5);
    }

    carrier.Light.readColor(
        reading.light_color.r,
        reading.light_color.g,
        reading.light_color.b, 
        re.brightness
    );

    // https://docs.arduino.cc/libraries/arduino_hts221/
    reading.temperature carrier.Env.readTemperature();
    reading.humidity = carrier.Env.readHumidity();
    // https://docs.arduino.cc/libraries/arduino_lps22hb/
    reading.air_pressure = carrier.Pressure.readPressure();

    return reading;
}

struct color {
    r int;
    g int;
    b int;
}

struct Reading {
    temperature float;
    humidity float;
    air_pressure float;
    brightness int;
    light_color: color;
}
