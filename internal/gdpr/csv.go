package gdpr

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
)

// writeCSV serialises a slice of row maps to CSV. Column order is
// alphabetical for deterministic output across runs.
func writeCSV(w io.Writer, rows []map[string]interface{}) error {
	if len(rows) == 0 {
		return nil
	}

	// Collect sorted column names from first row.
	cols := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		cols = append(cols, k)
	}
	sort.Strings(cols)

	cw := csv.NewWriter(w)
	if err := cw.Write(cols); err != nil {
		return fmt.Errorf("csv header: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(cols))
		for i, col := range cols {
			v := row[col]
			if v == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("csv row: %w", err)
		}
	}
	cw.Flush()
	return cw.Error()
}
