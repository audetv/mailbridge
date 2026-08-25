package preprocessor

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// extractXlsxText извлекает текст из XLSX-файла.
func extractXlsxText(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var text strings.Builder
	sheets := f.GetSheetList()
	if len(sheets) > MaxXLSXSheets {
		sheets = sheets[:MaxXLSXSheets]
	}

	for _, sheet := range sheets {
		text.WriteString("=== Лист: " + sheet + " ===\n")
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}

		if len(rows) > MaxXLSXRows {
			fmt.Fprintf(&text, "[Показаны первые %d из %d строк]\n", MaxXLSXRows, len(rows))
			rows = rows[:MaxXLSXRows]
		}

		for _, row := range rows {
			text.WriteString(strings.Join(row, " | "))
			text.WriteString("\n")
		}
	}

	return truncateText(text.String()), nil
}
