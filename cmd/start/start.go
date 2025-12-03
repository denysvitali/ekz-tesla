package start

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
)

var (
	boxID       string
	connectorID int
)

var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start charging at a charging station",
	Long: `Start a charging session at the specified charging station.
If no box ID or connector ID is provided, uses values from configuration.`,
	Example: `  # Start charging using config values
  ekz-tesla start

  # Start charging at a specific box and connector
  ekz-tesla start --box-id CH-EKZ-E001234 --connector-id 1`,
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
		log.Debugf("Starting charge at box %s, connector %d", boxID, connectorID)

		remoteStart, err := client.RemoteStart(boxID, connectorID)
		if err != nil {
			return fmt.Errorf("failed to start charging: %w", err)
		}

		if remoteStart != nil {
			fmt.Printf("✅ Charging started successfully\n")
			log.Debugf("Remote start response: %+v", remoteStart)
		} else {
			fmt.Printf("✅ Charging command sent\n")
		}

		return nil
	},
}

func init() {
	StartCmd.Flags().StringVar(&boxID, "box-id", "", "Charging station box ID")
	StartCmd.Flags().IntVar(&connectorID, "connector-id", 0, "Connector ID")

	root.RootCmd.AddCommand(StartCmd)
}