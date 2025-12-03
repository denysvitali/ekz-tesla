package list

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
	"github.com/denysvitali/ekz-tesla/ekz"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available charging stations",
	Long: `List all charging stations associated with your EKZ account, showing their status and availability.

The output shows the Box ID and Connector ID needed for the start/stop commands.`,
	Example: `  # List all charging stations
  ekz-tesla list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := root.GetClient()
		if client == nil {
			return fmt.Errorf("EKZ client not initialized")
		}

		chargingStations, err := client.GetUserChargingStations()
		if err != nil {
			return fmt.Errorf("failed to get user charging stations: %w", err)
		}

		if len(chargingStations) == 0 {
			fmt.Println("No charging stations found.")
			return nil
		}

		printChargingStations(chargingStations)
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(ListCmd)
}

// printChargingStations prints the charging stations using lipgloss's table
func printChargingStations(stations []ekz.ChargingStation) {
	var rows [][]string
	for _, s := range stations {
		for _, box := range s.ChargeBoxes {
			onlineStatus := "❌"
			if box.Online {
				onlineStatus = "✅"
			}

			// If there are connectors, show each one as a separate row
			if len(box.Connectors) > 0 {
				for _, conn := range box.Connectors {
					status := conn.ConnectorStatus
					if status == "" {
						status = conn.Status
					}
					if status == "" {
						status = "-"
					}

					plugType := conn.PlugType
					if plugType == "" {
						plugType = "-"
					}

					rows = append(rows, []string{
						box.ChargeBoxID,
						fmt.Sprintf("%d", conn.ConnectorID),
						box.ChargeBoxName,
						plugType,
						status,
						onlineStatus,
					})
				}
			} else {
				// Fallback: show box-level info if no connectors available
				status := box.ConnectorStatus
				if status == "" {
					status = box.ChargingProcessStatus
				}
				if status == "" {
					status = "-"
				}

				plugType := box.PlugType
				if plugType == "" {
					plugType = "-"
				}

				rows = append(rows, []string{
					box.ChargeBoxID,
					"-",
					box.ChargeBoxName,
					plugType,
					status,
					onlineStatus,
				})
			}
		}
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		Headers("BOX ID", "CONN", "NAME", "PLUG TYPE", "STATUS", "ONLINE").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).PaddingLeft(1).PaddingRight(1)
			}
			baseStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

			// Center align connector ID, status, and online columns
			if col == 1 || col >= 4 {
				return baseStyle.AlignHorizontal(lipgloss.Center)
			}
			return baseStyle
		}).
		Rows(rows...)

	fmt.Println(t)
}