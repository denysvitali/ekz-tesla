package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/denysvitali/ekz-tesla/ekz"
)

// printChargingStations prints the charging stations using lipgloss's table
func printChargingStations(stations []ekz.ChargingStation) {
	var rows [][]string
	for _, s := range stations {
		rows = append(rows, []string{
			s.ChargeBoxes[0].ChargeBoxID,
			s.ChargeBoxes[0].ChargeBoxName,
			s.ChargeBoxes[0].ChargingProcessStatus,
			s.ChargeBoxes[0].ConnectorStatus,
			fmt.Sprintf("%v", s.ChargeBoxes[0].Online),
		})
	}
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99"))).
		Headers("ID", "NAME", "STATUS", "CONNECTOR STATUS", "ONLINE").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == 0 {
				return lipgloss.NewStyle().Bold(true).PaddingLeft(1).PaddingRight(1)
			}
			baseStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)
			if col > 1 {
				return baseStyle.AlignHorizontal(lipgloss.Center)
			}
			return baseStyle
		}).
		Rows(rows...)

	fmt.Println(t)
}

func printLiveData(liveData *ekz.LiveDataResponse) {
	fmt.Printf("Status: %v\n", liveData.Status)
	fmt.Printf("Power: %0.2f\n", liveData.Power)
	fmt.Printf("Energy: %.2f\n", liveData.ChargedEnergy)
	fmt.Println()
}
