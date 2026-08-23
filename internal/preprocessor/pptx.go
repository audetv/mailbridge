package preprocessor

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// extractPptxText извлекает текст из PPTX-файла.
// PPTX — это ZIP-архив с XML-слайдами.
func extractPptxText(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var result strings.Builder
	slideNum := 0

	for _, file := range reader.File {
		// Слайды: ppt/slides/slide1.xml, slide2.xml, ...
		if !strings.HasPrefix(file.Name, "ppt/slides/slide") {
			continue
		}
		if !strings.HasSuffix(file.Name, ".xml") {
			continue
		}

		slideNum++
		fmt.Fprintf(&result, "=== Слайд %d ===\n", slideNum)

		content, err := extractTextFromSlideXML(file)
		if err == nil {
			result.WriteString(content)
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// extractTextFromSlideXML извлекает текст из XML слайда.
func extractTextFromSlideXML(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "t" {
				// Текстовый элемент <a:t>
				var text string
				if err := decoder.DecodeElement(&text, &elem); err == nil {
					result.WriteString(text)
				}
			}
		}
	}

	return result.String(), nil
}
