package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var headerStyle = lipgloss.NewStyle().Bold(true)

// renderJSON writes v as indented JSON.
func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// renderTable writes a lipgloss-styled table.
func renderTable(w io.Writer, headers []string, rows [][]string) {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})
	_, _ = fmt.Fprintln(w, t)
}
