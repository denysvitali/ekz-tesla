package stop

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
)

var (
	boxID       string
	connectorID int
)

var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop charging at a charging station",
	Long: `Stop an active charging session at the specified charging station.
If no box ID or connector ID is provided, uses values from configuration.`,
	Example: `  # Stop charging using config values
  ekz-tesla stop

  # Stop charging at a specific box and connector
  ekz-tesla stop --box-id CH-EKZ-E001234 --connector-id 1`,
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
		log.Debugf("Stopping charge at box %s, connector %d", boxID, connectorID)

		remoteStop, err := client.RemoteStop(boxID, connectorID)
		if err != nil {
			return fmt.Errorf("failed to stop charging: %w", err)
		}

		if remoteStop != nil {
			fmt.Printf("✅ Charging stopped successfully\n")
			log.Debugf("Remote stop response: %+v", remoteStop)
		} else {
			fmt.Printf("✅ Stop command sent\n")
		}

		return nil
	},
}

func init() {
	StopCmd.Flags().StringVar(&boxID, "box-id", "", "Charging station box ID")
	StopCmd.Flags().IntVar(&connectorID, "connector-id", 0, "Connector ID")

	root.RootCmd.AddCommand(StopCmd)
}