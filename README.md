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
Demeter is primarily focused on "garden" conditions, or specific diagnostics, though plot- or plant-specific devices may be introduced if the additional granularity is justified.

| Sensor | Type | Unit |
| - | - | - |
| Humidity | `number` | g/m<sup>3</sup> |
| Temperature | `number` | C |
| Air Pressure | `number` | Pa |
| Light (Brightness) | `number` | lx |
| Soil Moisture | `number` | *relative* | % |


It's meant primarily as an intermediary between the measuring device(s) and any services that want to consume the data.
As such, its footprint will likely remain small, with additional functionality introduced through integrations.

| Iteration | Device | Service |
| - | - | - |
| 0 | Tracks conditions for one "garden" | Writes readings to a local DB |
| 1 | + (optionally) specific "plots" | - |
| 2 | - | Calls registered webhook |
| 3 | - | Supports multiple clients |


### Philosophy
- Software can empower individuals to enrich their lives and strengthen their communities.
    - Good software has a clear scope, driven by user needs.
- Automation can be liberating, but 
- it's helpful to have data and automate things, but our obsession with efficiency has a horrific human and environmental cost
  - we should be mindful of how we're using resources: just because chips or compute are cheap to us, doesn't mean they're costless 
- 

## Architecture
In its simplest form, Demeter consists of:
- a Device that collects readings from sensors and makes remote calls to,
- a Service that can then persist (and/or relay) that data to a consumer

### Device
At a minimum, Demeter requries a board with the relevant environmental sensors and remote calling capabilities.

The MVP is the prefabricated Arduino Opla board, which includes all sensors, and an SDK to read and post data over WiFi. The board design can be optimized for cost and accessibility though, and subsequent iterations will attempt to address these issues.

#### Client
*Regularly dispatch data updates from connected sensors.*

This could just be a library that Arduino users can import and use manually, but in the long term, packaging boards with firmware improves accessibility (and makes Demeter easier to commodify).

- Read from sensors and call remote service
- Persist data locally short-term for retries?

### Service
*Manage data persistence, provide data access, and dispatch alerts.*

Additional clarity is needed regarding how the data should be used, or made available. Regardless, business logic shouldn't block data ingestion. Maximizing throughput is a priority for this layer.

There are a few possible features with a clear value proposition:
- Persist data
    - Used to determine alert thresholds, but also possibly analytics, via dashboard and/or API
- Dispatch alerts (if values change dramatically, or exceed fixed or configured thresholds).

Writing to a local SQLite database is a solid short-term solution: its both fairly simple and pretty performant/scalable. Postgres seems excessive without a more concrete idea of how the data will be used (e.g., will Demeter include a dashboard, or just dispatch events).

*How many of these questions should block development altogether?"

#### Endpoint

`POST /readings/{sensor_id}`

| Property | Type | Unit |
| - | - | - |
| timestamp | `number` | Unix? UTC? |
| humidity | `number` | g/m<sup>3</sup> |
| temperature | `number` | C |
| air_pressure | `number` | Pa |
| brightness | `number` | lx |
| soil_moisture | `number` | *relative, optional* | % |
