# demeter
Smart plant monitoring

**Demeter is a lightweight solution for monitoring and recording local growing conditions remotely.**
You bring the questions (and the plants), and Demeter brings the data--without the baggage of vendor lock-in or massive overhead.

Demeter tracks serial environmental data to support a few common use-cases:
- Planning: Place plants where you know they'll thrive.
- Diagnostics: Is a struggling plant not getting enough water or light?
- Automation: Tune irrigation based on weather conditions.
- Monitoring: Just in case the drip irrigation fails.

Demeter is primarily focused on "garden" conditions, or specific diagnostics.
- don't need to measure temperature for every plant individually
- future:
  - add plot- or plant-level nodes for additional granularity 
  - integrate with systems for garden management and plant cultivation

| Sensor | Type | Unit |
| - | - | - |
| Humidity | `number` | g/m<sup>3</sup> |
| Temperature | `number` | C |
| Air Pressure | `number` | Pa |
| Light (Brightness) | `number` | lx |
| Soil Moisture | `number` | *relative* | % |

## Philosophy
- software can empower individuals to enrich their lives and strengthen their communities
- it's helpful to have data and automate things, but our obsession with efficiency has a horrific human and environmental cost
  - we should be mindful of how we're using resources: just because chips or compute are cheap to us, doesn't mean they're costless 
- good software has a clear scope, driven by user needs

## Architecture

### Hardware
- could start with the arduino (though not currently available)
- otherwise need microcontroller, wifi chip, etc

### Client
*Regularly dispatch data updates from connected sensors.*
- Read from sensors (native board API?)
- HTTP Client
- Queue/Retries

- maybe i'm overthinking this, but it seems like this should be as slim as possible
    - it's not meant to do much of anything, and we want as much space as possible for a retry queue

### Services
*Manage data persistence, provide data access, and dispatch alerts.*

#### Record Tick

- `POST` `/tick/{sensor_id}`

- form-url-encode:

| Property | Name | Type | Unit |
| - | - | - | - |
| sid | sensor_id | `string` | - |
| t | timestamp | `number` | unix? UTC |
| hum | humidity | `number` | g/m<sup>3</sup> |
| tmp | temperature | `number` | C |
| psr | air_pressure | `number` | Pa |
| brt | brightness | `number` | lx |
| smo | soil_moisture | `number \| undefined` | *relative* | % |

#### Dashboard

| Property | Type | Unit |
| - | - | - |
| timestamp | `number` | unix? UTC? |
| humidity | `number` | g/m<sup>3</sup> |
| temperature | `number` | C |
| air_pressure | `number` | Pa |
| brightness | `number` | lx |
| soil_moisture | `number \| undefined` | *relative* | % |

### Persistence
*Store data for short-term monitoring and long-term trend analysis.*

- if the database is only backing a single client, sqlite should work fine for at least the first few years of continuous (hourly) data
- how painful is it to migrate from sqlite to postgres?
- realistically, to what extent is persistence even in scope?
    - having a short-term window seems fine for now, and data could get pushed elsewhere

## Questions

- Should values be stored as one unit and converted on display (as needed), or should they be converted and stored in parallel?
  - Storing multiple units doubles+ the persistence footprint, but converting on read could be fragile? 