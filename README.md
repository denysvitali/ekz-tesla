# ekz-tesla

A Go application to automatically manage charging sessions for Tesla vehicles connected to EKZ home chargers.

## Description

ekz-tesla helps Tesla owners automate their charging experience with EKZ home chargers by:

- Starting/stopping charging sessions manually
- Listing charging station information
- Displaying live charging data
- Automatically starting charging when your Tesla is:
    - Plugged in
    - Below your desired charge level
    - Located near your charging station
- Scheduling automatic charging using cron expressions

## Installation

### Prerequisites

- Go 1.18 or higher
- Access to an EKZ charging station
- [TeslaMate](https://github.com/teslamate-org/teslamate) [API access](https://github.com/tobiasehlert/teslamateapi)

### Building from source

```bash
git clone https://github.com/denysvitali/ekz-tesla.git
cd ekz-tesla
go build -o ekz-tesla .
```

## Configuration

Create a config file (JSON format) with the following structure:

```json
{
  "charging_station": {
    "latitude": 47.123456,
    "longitude": 8.123456,
    "box_id": "YOUR_BOX_ID",
    "connector_id": 1
  },
  "ekz": {
    "username": "your_username",
    "password": "your_password"
  }
}
```

## Usage

### Basic Commands

Start charging:
```bash
./ekz-tesla -c config.json start
```

Stop charging:
```bash
./ekz-tesla -c config.json stop
```

List charging stations:
```bash
./ekz-tesla -c config.json list
```

Monitor live charging data:
```bash
./ekz-tesla -c config.json live-data
```

### Automatic Charging

Start charging automatically when conditions are met:

```bash
./ekz-tesla -c config.json autostart --car-id 1 --teslamate-api-url http://teslamate-api:8080 --maximum-charge 90
```

### Scheduled Charging

Set up a recurring schedule for automatic charging:

```bash
./ekz-tesla -c config.json scheduled-autostart --car-id 1 --teslamate-api-url http://teslamate-api:8080 --maximum-charge 90 --cronjob-line "0 22 * * *"
```

### Options

- `-c, --config`: Path to configuration file
- `--log-level`: Set logging level (default: info)
- `--maximum-charge`: Maximum battery charge percentage (default: 90)

## License

MIT License - See `LICENSE.txt` for details.
