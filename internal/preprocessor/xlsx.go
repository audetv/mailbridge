package preprocessor

import (
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
	for _, sheet := range f.GetSheetList() {
		text.WriteString("=== Лист: " + sheet + " ===\n")
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			text.WriteString(strings.Join(row, " | "))
			text.WriteString("\n")
		}
	}
	return text.String(), nil
}
