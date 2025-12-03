package livedata

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
	"github.com/denysvitali/ekz-tesla/ekz"
)

const (
	maxHistorySize = 30 // Number of data points to keep for sparkline
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1)

	sparklineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	chargingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

	availableStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			MarginTop(1)
)

// Sparkline characters (from lowest to highest)
var sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// History tracking
type historyPoint struct {
	power  float64
	energy float64
	time   time.Time
}

var history []historyPoint

var (
	boxID       string
	connectorID int
	interval    int
	once        bool
)

var LiveDataCmd = &cobra.Command{
	Use:   "live-data",
	Short: "Display live charging data",
	Long: `Display live data from the charging station including power consumption,
energy delivered, and charging status. By default, updates every 5 seconds.

Shows a sparkline graph of power consumption over time.`,
	Example: `  # Display live data using config values
  ekz-tesla live-data

  # Display live data for a specific box and connector
  ekz-tesla live-data --box-id CH-EKZ-E001234 --connector-id 1

  # Get data once and exit
  ekz-tesla live-data --once

  # Update every 10 seconds
  ekz-tesla live-data --interval 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := root.GetClient()
		if client == nil {
			return fmt.Errorf("EKZ client not initialized")
		}

		cfg := root.GetConfig()
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		// Use provided values or fall back to config
		if boxID == "" {
			boxID = cfg.ChargingStation.BoxId
		}
		if connectorID == 0 {
			connectorID = cfg.ChargingStation.ConnectorId
		}

		// Validate required parameters
		if boxID == "" {
			return fmt.Errorf("box ID is required (use --box-id or set in config)")
		}
		if connectorID == 0 {
			return fmt.Errorf("connector ID is required (use --connector-id or set in config)")
		}

		log := root.GetLogger()
		log.Debugf("Getting live data for box %s, connector %d", boxID, connectorID)

		// Reset history for new session
		history = nil

		// Get live data once or continuously
		for {
			liveData, err := client.GetLiveData(boxID, connectorID, "")
			if err != nil {
				return fmt.Errorf("failed to get live data: %w", err)
			}

			// Record history
			recordHistory(liveData)

			printLiveData(liveData)

			if once {
				break
			}

			time.Sleep(time.Duration(interval) * time.Second)
		}

		return nil
	},
}

func init() {
	LiveDataCmd.Flags().StringVar(&boxID, "box-id", "", "Charging station box ID")
	LiveDataCmd.Flags().IntVar(&connectorID, "connector-id", 0, "Connector ID")
	LiveDataCmd.Flags().IntVar(&interval, "interval", 5, "Update interval in seconds")
	LiveDataCmd.Flags().BoolVar(&once, "once", false, "Get data once and exit")

	root.RootCmd.AddCommand(LiveDataCmd)
}

func recordHistory(liveData *ekz.LiveDataResponse) {
	point := historyPoint{
		power:  liveData.Power,
		energy: liveData.ChargedEnergy,
		time:   time.Now(),
	}

	history = append(history, point)

	// Keep only the last N points
	if len(history) > maxHistorySize {
		history = history[1:]
	}
}

func printLiveData(liveData *ekz.LiveDataResponse) {
	// Clear screen for continuous updates (unless running once)
	if !once {
		fmt.Print("\033[H\033[2J")
	}

	// Title
	fmt.Println(titleStyle.Render("LIVE CHARGING DATA"))

	// Build the data table
	rows := [][]string{
		{"Status", getStyledStatus(liveData.Status)},
		{"Power", fmt.Sprintf("%.2f kW", liveData.Power)},
		{"Energy", fmt.Sprintf("%.2f kWh", liveData.ChargedEnergy)},
		{"Updated", time.Now().Format("15:04:05")},
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			baseStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
			if col == 0 {
				return baseStyle.Foreground(lipgloss.Color("241"))
			}
			return baseStyle.Bold(true)
		}).
		Rows(rows...)

	fmt.Println(t)

	// Show power trend if we have history
	if len(history) > 1 && !once {
		fmt.Println()
		printPowerTrend()
	}

	if !once {
		fmt.Println(hintStyle.Render("Press Ctrl+C to exit"))
	}
}

func printPowerTrend() {
	if len(history) < 2 {
		return
	}

	// Extract power values
	values := make([]float64, len(history))
	for i, h := range history {
		values[i] = h.power
	}

	// Calculate stats
	minVal, maxVal, avgVal := calculateStats(values)

	// Generate sparkline
	sparkline := generateSparkline(values)

	// Calculate time range
	duration := history[len(history)-1].time.Sub(history[0].time)

	// Print trend section
	fmt.Println(dimStyle.Render("Power Trend"))
	fmt.Println(sparklineStyle.Render(sparkline))
	fmt.Println(dimStyle.Render(fmt.Sprintf(
		"Min: %.1f kW  Max: %.1f kW  Avg: %.1f kW  (%s)",
		minVal, maxVal, avgVal, formatDuration(duration),
	)))
}

func generateSparkline(values []float64) string {
	if len(values) == 0 {
		return ""
	}

	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Handle case where all values are the same
	valueRange := maxVal - minVal
	if valueRange == 0 {
		valueRange = 1
	}

	var sb strings.Builder
	for _, v := range values {
		// Normalize to 0-7 range for sparkline characters
		normalized := (v - minVal) / valueRange
		index := int(normalized * float64(len(sparklineChars)-1))
		if index >= len(sparklineChars) {
			index = len(sparklineChars) - 1
		}
		if index < 0 {
			index = 0
		}
		sb.WriteRune(sparklineChars[index])
	}

	return sb.String()
}

func calculateStats(values []float64) (min, max, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	min = math.MaxFloat64
	max = -math.MaxFloat64
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg = sum / float64(len(values))
	return min, max, avg
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func getStyledStatus(status string) string {
	switch status {
	case "CHARGING":
		return chargingStyle.Render("⚡ Charging")
	case "AVAILABLE":
		return availableStyle.Render("✓ Available")
	case "OCCUPIED":
		return warningStyle.Render("⏸ Occupied")
	case "UNAVAILABLE":
		return errorStyle.Render("✗ Unavailable")
	case "PREPARING":
		return warningStyle.Render("↻ Preparing")
	case "FINISHING":
		return warningStyle.Render("⏳ Finishing")
	default:
		return status
	}
}
