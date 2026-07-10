package cmd

import (
	"fmt"
	"strings"
)

type labeledValue struct {
	label string
	value string
}

// printTable renders a fixed-width text table with a header row and separator.
// By default only interior columns are padded; the last column is left unpadded.
func printTable(prefix string, headers []string, rows [][]string, opts ...tableOption) {
	if len(headers) == 0 {
		return
	}

	cfg := tableConfig{prefix: prefix, padLastColumn: false}
	for _, opt := range opts {
		opt(&cfg)
	}

	widths := columnWidths(headers, rows)
	for i, min := range cfg.minColumnWidths {
		if i < len(widths) && min > widths[i] {
			widths[i] = min
		}
	}

	fmt.Println(prefix + formatTableCells(widths, headers, cfg.padLastColumn, cfg.tightColumns))

	sepLen := tableSeparatorWidth(widths, cfg.tightColumns)
	fmt.Println(prefix + strings.Repeat("-", sepLen))

	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		fmt.Println(prefix + formatTableCells(widths, cells, cfg.padLastColumn, cfg.tightColumns))
	}
}

type tableConfig struct {
	prefix          string
	padLastColumn   bool
	tightColumns    bool
	minColumnWidths []int
}

type tableOption func(*tableConfig)

func withPadAllColumns() tableOption {
	return func(cfg *tableConfig) {
		cfg.padLastColumn = true
	}
}

func withTightColumns() tableOption {
	return func(cfg *tableConfig) {
		cfg.tightColumns = true
	}
}

func withMinColumnWidths(widths []int) tableOption {
	return func(cfg *tableConfig) {
		cfg.minColumnWidths = widths
	}
}

func formatTableCells(widths []int, cells []string, padLastColumn, tight bool) string {
	parts := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		if i == len(widths)-1 && !padLastColumn {
			parts[i] = cell
		} else {
			parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
		}
	}
	sep := " "
	if tight {
		sep = ""
	}
	return strings.Join(parts, sep)
}

func printLabelValueTable(labelHeader, valueHeader string, rows ...labeledValue) {
	if len(rows) == 0 {
		return
	}

	tableRows := make([][]string, len(rows))
	for i, row := range rows {
		tableRows[i] = []string{row.label, row.value}
	}
	printTable("", []string{labelHeader, valueHeader}, tableRows)
}

func printLabeledValues(rows ...labeledValue) {
	printLabelValueTable("FIELD", "VALUE", rows...)
}

// printContextFields renders "Label: value" lines for context add/show output.
func printContextFields(prefix string, rows ...labeledValue) {
	if len(rows) == 0 {
		return
	}

	maxLabel := 0
	for _, row := range rows {
		if n := len(row.label) + 1; n > maxLabel {
			maxLabel = n
		}
	}

	for _, row := range rows {
		fmt.Printf("%s%-*s %s\n", prefix, maxLabel, row.label+":", row.value)
	}
}

func columnWidths(headers []string, rows [][]string) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}
	return widths
}

func tableSeparatorWidth(widths []int, tight bool) int {
	if len(widths) == 0 {
		return 0
	}
	total := 0
	for _, width := range widths {
		total += width
	}
	if !tight {
		total += len(widths) - 1
	}
	return total
}

func separatorWidth(widths []int) int {
	return tableSeparatorWidth(widths, false)
}
