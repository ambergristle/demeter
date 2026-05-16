# demeter
Smart plant monitoring

**Demeter is a lightweight solution for monitoring and recording local growing conditions remotely.**
You bring the questions (and the plants), and Demeter brings the data--without the baggage of vendor lock-in or massive overhead.

Demeter tracks serial environmental data to support a few common use-cases:
- Planning: Place plants where you know they'll thrive.
- Diagnostics: Is a struggling plant not getting enough water or light?
- Automation: Tune irrigation based on weather conditions.
- Monitoring: Just in case the drip irrigation fails.

### Roadmap
Demeter is focused on "garden" conditions, or specific diagnostics, though plot- or plant-specific devices may be introduced if the additional granularity is justified.

| Sensor | Type | Unit |
| - | - | - |
| Humidity | `number` | g/m<sup>3</sup> |
| Temperature | `number` | C |
| Air Pressure | `number` | Pa |
| Light (Brightness) | `number` | lx |
| Soil Moisture | `number` | *relative* | % |

There are a number of other inputs that could be collected and/or processed, namely:
- Wind
- Rain
- pH
- Nutrient levels
- Image analysis (tracking color, growth, etc.)

These all require additional hardware though, and image analysis explodes project complexity. Support may be considered post-MVP, but is out of scope at this time.

Demeter is an intermediary between the measuring device(s) and any services that want to consume the data.
While the device itself is capable of making an HTTP request directly to consumers, handling request failures is much more challenging in a memory-constrained environment.
**Demeter's core thesis is that it's cheaper to get data off the device ASAP, and ensure it's captured, regardless of how it's consumed.**

In light of its goals, Demeter's footprint will likely remain small, with additional functionality introduced through integrations.

| Iteration | Introduces |
| - | - |
| 0 | Service that validates Reading payload and makes signed call to registered webhook. |
| 1 | Basic security and retry logic |
| 2 | Queue or other overflow handler |
| 3 | Logging and monitoring |
| 4 | Accounts and registration, advanced security |

### Philosophy

**Software should empower individuals to enrich their lives and strengthen their communities.** Both in its intended use and its development, Demeter strives to increase knowledge, understanding, and collaboration.

**Good software has a clear scope, driven by user needs.** Above all, Demeter is a public service. Its success is measured in the value it brings to users: what it enables them to create or share.

**Value to one community shouldn't come at the expense of another.** Automation can be liberating, but we must be mindful of technology's "hidden" human and environmental costs.

## Architecture
In its simplest form, Demeter consists of:
- a Device that collects readings from sensors and makes remote calls to,
- a Service that can then persist (and/or relay) that data to a consumer

### Device
At a minimum, Demeter requries a board with the relevant environmental sensors and remote calling capabilities.

The MVP is the prefabricated [Arduino Opla](https://opla.arduino.cc/) board, which includes all sensors, and an SDK to read and post data over WiFi.

> *The board was on-hand, and does all the basics. This is not necessarily an endorsement of Arduino or the kit.*

- [MKR WiFi 1010](https://docs.arduino.cc/hardware/mkr-wifi-1010/): WiFi + Bluetooth-capable microcontroller board
- [HTS221](https://www.st.com/en/mems-and-sensors/hts221.html): Temperature and Humidity
- [LPS22HB](https://www.st.com/en/mems-and-sensors/lps22hb.html): Air Pressure (and Temperature)
- [APDS-9960](https://www.adafruit.com/product/3595): Light, proximity, and gesture sensing

The board design can be optimized for cost and accessibility though, and subsequent iterations will attempt to address these issues.

#### Client
*Regularly dispatch data updates from connected sensors.*

This could just be a library that Arduino users can import and use manually, but in the long term, packaging boards with firmware improves accessibility (and makes Demeter easier to commodify).

- Read from sensors and call remote service.
- Persist data locally short-term for retries?

Values are read in from sensors using [Arduino lirbaries](https://docs.arduino.cc/tutorials/mkr-iot-carrier/mkr-iot-carrier-01-technical-reference/).

| Property | Type | Unit | Range |
| - | - | - | - |
| timestamp | `int` | Unix | - |
| humidity | `float` | g/m<sup>3</sup> | 0-100% |
| temperature | `float` | °C | 40-120°C |
| air_pressure | `float` | kPa | absolute range 260-1260hPa |
| brightness | `int` | lx | - |

*The only limits to brightness values appear to be memory-related. I haven't found a spec that clearly addresses this, but it seems capable of +50k lx (a sunny day).*

### Service
*Relay readings to registered callback. And more?*

Demeter's backend is a minimal [Golang](https://golang.org/) server that exposes an endpoint to ingest readings.
**Valid readings are forwarded to the registered callback with a signed request.**

Additional clarity is needed regarding how the data should be used, or made available. Regardless, business logic shouldn't block data ingestion. Maximizing throughput is a priority for this layer.

#### `POST /readings/{sensor_id}`

The Readings endpoint forwards valid reading data to a registered callback.

*Value formats must be confirmed.*

| Property | Type | Unit |
| - | - | - |
| timestamp | `number` | Unix? UTC? |
| humidity | `number` | g/m<sup>3</sup> |
| temperature | `number` | C |
| air_pressure | `number` | kPa |
| brightness | `number` | lx |

The callback request is secured with an [HMAC signature](https://en.wikipedia.org/wiki/HMAC).

| Header | Value |
| - | - |
| `content-type` | `application/x-www-form-urlencoded` |
| `x-dmtr-timestamp` | Unix UTC timestamp |
| `x-dmtr-signature` | HMAC signature (body + timestamp) |

## Verify callback requests

1. Validate request timestamp
```go
// Get timestamp from Header, and parse to int
requestTimestamp := r.Header.Get("x-dmtr-timestamp")
timestamp, err := strconv.ParseInt(requestTimestamp, 10, 64)
// >> 1531420618
if err != nil {
    // Timestamp is invalid: reject
}

// Verify timestamp happened within 5 minutes of local time
now := time.Now()
stampTime := time.Unix(stamp, 0)
timeDifference := now.Sub(stampTime).Abs().Seconds()
if timeDifference > 60 * 5 {
    // Possible replay attack: reject
}
```

2. Construct the signature
```go
bodyBytes, err := io.ReadAll(resp.Body)
if err != nil {
    // Bad request: reject
}

signatureBase := "v0:" + timestamp + ":" + string(bodyBytes)
// >> v0:1531420618:timestamp=1778870328&humidity=90&temperature=27...

// Secret comes from somewhere
mac := hmac.New(sha256.New, secret)
mac.Write([]byte(signatureBase))

signature := "v0=" + hex.EncodeToString(mac.Sum(nil))
// >> v0=a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503
```

4. Compare signatures
```go
requestSignature := r.Header.Get("x-dmtr-signature")
// >> v0=a2114d57b48eac39b9ad189dd8316235a7b4a8d21a10bd27519666489c69b503

// Compare signature bytes without leaking timing info
if !hmac.Equal([]byte(signature), []byte(requestSignature)) {
    // Invalid signature: reject
}
```
