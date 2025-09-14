package livedata

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
	"github.com/denysvitali/ekz-tesla/ekz"
)

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
energy delivered, and charging status. By default, updates every 5 seconds.`,
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

		// Get live data once or continuously
		for {
			liveData, err := client.GetLiveData(boxID, connectorID, "")
			if err != nil {
				return fmt.Errorf("failed to get live data: %w", err)
			}

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

func printLiveData(liveData *ekz.LiveDataResponse) {
	// Clear screen for continuous updates (unless running once)
	if !once {
		fmt.Print("\033[H\033[2J")
	}

	fmt.Println("╭──────────────────────────────────╮")
	fmt.Println("│       LIVE CHARGING DATA         │")
	fmt.Println("├──────────────────────────────────┤")
	fmt.Printf("│ Status:  %-23s │\n", getStatusEmoji(liveData.Status))
	fmt.Printf("│ Power:   %-23.2f kW │\n", liveData.Power)
	fmt.Printf("│ Energy:  %-23.2f kWh│\n", liveData.ChargedEnergy)
	fmt.Println("├──────────────────────────────────┤")
	fmt.Printf("│ Updated: %-23s │\n", time.Now().Format("15:04:05"))
	fmt.Println("╰──────────────────────────────────╯")

	if !once {
		fmt.Println("\nPress Ctrl+C to exit")
	}
}

func getStatusEmoji(status string) string {
	switch status {
	case "CHARGING":
		return "⚡ Charging"
	case "AVAILABLE":
		return "✅ Available"
	case "OCCUPIED":
		return "🔌 Occupied"
	case "UNAVAILABLE":
		return "❌ Unavailable"
	case "PREPARING":
		return "🔄 Preparing"
	case "FINISHING":
		return "⏳ Finishing"
	default:
		return status
	}
}