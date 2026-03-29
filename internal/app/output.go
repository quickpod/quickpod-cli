package app

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func PrintJSON(data any) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(encoded))
	return err
}

func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}

func PrintSection(title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	fmt.Fprintln(os.Stdout, title)
}

func PrintKeyValues(values map[string]string) {
	rows := make([][]string, 0, len(values))
	for key, value := range values {
		rows = append(rows, []string{key, value})
	}
	PrintTable([]string{"KEY", "VALUE"}, rows)
}