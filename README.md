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
- Smart scheduling based on electricity tariffs (charges only during low tariff periods)
- ⚠️ **DEPRECATED**: Manual scheduling using cron expressions (use smart-autostart instead)

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

```yaml
username: foo@example.com
password: your-password
charging_station:
  box_id:  22222222
  connector_id: 1
  latitude: 47.123456
  longitude: 8.123456
```

## Usage

### Basic Commands

Start charging:
```bash
./ekz-tesla -c config.yaml start
```

Stop charging:
```bash
./ekz-tesla -c config.yaml stop
```

List charging stations:
```bash
./ekz-tesla -c config.yaml list
```

Monitor live charging data:
```bash
./ekz-tesla -c config.yaml live-data
```

### Automatic Charging

Start charging automatically when conditions are met:

```bash
./ekz-tesla -c config.yaml autostart --car-id 1 --teslamate-api-url http://teslamate-api:8080 --maximum-charge 90
```

### Smart Scheduling (Recommended)

**NEW**: Automatically charge during low tariff periods based on predefined schedules:

```bash
./ekz-tesla -c config.yaml smart-autostart --car-id 1 --teslamate-api-url http://teslamate-api:8080 --maximum-charge 90
```

This intelligent scheduler:
- ⚡ Automatically starts charging only during low tariff periods
- 💰 Saves money by avoiding high tariff times (default: weekdays 7AM-8PM)
- 📅 Uses EKZ's standard schedule: Low tariff on weekends and weekday nights
- ⚙️ Customizable schedule via command-line arguments
- 🚀 Lightweight and efficient - no constant API polling

#### Custom Tariff Schedule

You can override the default schedule with custom high tariff times:

```bash
./ekz-tesla -c config.yaml smart-autostart \
  --car-id 1 \
  --teslamate-api-url http://teslamate-api:8080 \
  --maximum-charge 90 \
  --high-tariff-times "8:00-18:00:Mon,Tue,Wed,Thu,Fri" \
  --high-tariff-times "10:00-14:00:Sat,Sun"
```

**Time format**: `HH:MM-HH:MM:Weekdays` where weekdays are optional (Mon,Tue,Wed,Thu,Fri,Sat,Sun)

### Manual Scheduled Charging (DEPRECATED)

⚠️ **This approach is deprecated**. Use `smart-autostart` instead for better cost optimization.

Set up a recurring schedule for automatic charging using cron expressions:

```bash
./ekz-tesla -c config.yaml scheduled-autostart --car-id 1 --teslamate-api-url http://teslamate-api:8080 --maximum-charge 90 --cronjob-line "0 22 * * *"
```

**Why smart-autostart is better:**
- No need to manually calculate optimal charging times
- Automatically adapts to tariff changes
- Ensures you always charge at the cheapest rates

### Options

- `-c, --config`: Path to configuration file
- `--log-level`: Set logging level (default: info)
- `--maximum-charge`: Maximum battery charge percentage (default: 90)

## License

MIT License - See `LICENSE.txt` for details.
