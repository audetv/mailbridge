package preprocessor

import "github.com/nguyenthenguyen/docx"

// extractDocxText извлекает текст из DOCX-файла.
func extractDocxText(path string) (string, error) {
	doc, err := docx.ReadDocxFile(path)
	if err != nil {
		return "", err
	}
	defer doc.Close()

	text := doc.Editable().GetContent()
	return truncateText(text), nil
}
