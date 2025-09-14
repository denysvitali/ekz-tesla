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
	Long:  `List all charging stations associated with your EKZ account, showing their status and availability.`,
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
			status := box.ChargingProcessStatus
			if status == "" {
				status = "-"
			}

			connectorStatus := box.ConnectorStatus
			if connectorStatus == "" {
				connectorStatus = "-"
			}

			onlineStatus := "❌"
			if box.Online {
				onlineStatus = "✅"
			}

			rows = append(rows, []string{
				box.ChargeBoxID,
				box.ChargeBoxName,
				status,
				connectorStatus,
				onlineStatus,
			})
		}
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		Headers("ID", "NAME", "STATUS", "CONNECTOR", "ONLINE").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).PaddingLeft(1).PaddingRight(1)
			}
			baseStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

			// Center align status columns
			if col >= 2 {
				return baseStyle.AlignHorizontal(lipgloss.Center)
			}
			return baseStyle
		}).
		Rows(rows...)

	fmt.Println(t)
}