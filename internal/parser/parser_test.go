package parser_test

import (
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/parser"
)

// testParser создаёт FieldParser с тестовыми допустимыми значениями.
func testParser() *parser.FieldParser {
	return parser.NewFieldParser(
		map[string]bool{
			"bug": true, "feature": true, "support": true,
			"access": true, "seo": true, "content": true,
		},
		map[string]bool{
			"urgent": true, "high": true, "medium": true, "low": true,
		},
	)
}

func TestParse_AllFieldsRussian(t *testing.T) {
	p := testParser()

	text := `Проект: Отель
Тип: bug
Приоритет: high
Дедлайн: 2026-08-01
Исполнитель: Иванов

Описание: не работает бронирование номеров`

	result := p.Parse(text)

	if !result.HasFields {
		t.Fatal("expected HasFields=true")
	}
	if result.Project != "Отель" {
		t.Errorf("Project = %q, want %q", result.Project, "Отель")
	}
	if result.Type != "bug" {
		t.Errorf("Type = %q, want %q", result.Type, "bug")
	}
	if result.Priority != "High" {
		t.Errorf("Priority = %q, want %q", result.Priority, "High")
	}
	if result.Deadline != "2026-08-01" {
		t.Errorf("Deadline = %q, want %q", result.Deadline, "2026-08-01")
	}
	if result.Assignee != "Иванов" {
		t.Errorf("Assignee = %q, want %q", result.Assignee, "Иванов")
	}
	if result.Body != "Описание: не работает бронирование номеров" {
		t.Errorf("Body = %q", result.Body)
	}
}

func TestParse_AllFieldsEnglish(t *testing.T) {
	p := testParser()

	text := `Project: TRK
Type: support
Priority: medium
Deadline: tomorrow
Assignee: Petrov

Need to update the banner`

	result := p.Parse(text)

	if result.Project != "TRK" {
		t.Errorf("Project = %q", result.Project)
	}
	if result.Type != "support" {
		t.Errorf("Type = %q", result.Type)
	}
	if result.Priority != "Medium" {
		t.Errorf("Priority = %q", result.Priority)
	}
	if result.Deadline != "tomorrow" {
		t.Errorf("Deadline = %q", result.Deadline)
	}
	if result.Assignee != "Petrov" {
		t.Errorf("Assignee = %q", result.Assignee)
	}
}

func TestParse_MixedLanguages(t *testing.T) {
	p := testParser()

	text := `Проект: ТРК
Type: bug
Priority: urgent
Дедлайн: вчера

Срочная проблема`

	result := p.Parse(text)

	if result.Project != "ТРК" {
		t.Errorf("Project = %q", result.Project)
	}
	if result.Type != "bug" {
		t.Errorf("Type = %q", result.Type)
	}
	if result.Priority != "Urgent" {
		t.Errorf("Priority = %q", result.Priority)
	}
	if result.Deadline != "вчера" {
		t.Errorf("Deadline = %q", result.Deadline)
	}
}

func TestParse_NoFields(t *testing.T) {
	p := testParser()

	text := "Просто обычный текст письма без каких-либо полей"
	result := p.Parse(text)

	if result.HasFields {
		t.Error("expected HasFields=false")
	}
	if result.Body != text {
		t.Errorf("Body should be unchanged, got %q", result.Body)
	}
}

func TestParse_EmptyLineTerminatesHeader(t *testing.T) {
	p := testParser()

	text := `Проект: ТРК

Тип: bug
Текст после пустой строки`

	result := p.Parse(text)

	if result.Project != "ТРК" {
		t.Errorf("Project = %q", result.Project)
	}
	if result.Type != "" {
		t.Errorf("Type should be empty, got %q", result.Type)
	}
}

func TestParse_InvalidType(t *testing.T) {
	p := testParser()

	text := `Проект: ТРК
Тип: unknown_type
Приоритет: high

Текст`

	result := p.Parse(text)

	if result.Project != "ТРК" {
		t.Errorf("Project = %q", result.Project)
	}
	if result.Type != "" {
		t.Errorf("Type should be empty for invalid value, got %q", result.Type)
	}
	if result.Priority != "High" {
		t.Errorf("Priority = %q", result.Priority)
	}
}

func TestParse_InvalidPriority(t *testing.T) {
	p := testParser()

	text := `Тип: bug
Приоритет: максимальный

Текст`

	result := p.Parse(text)

	if result.Type != "bug" {
		t.Errorf("Type = %q", result.Type)
	}
	if result.Priority != "" {
		t.Errorf("Priority should be empty for invalid value, got %q", result.Priority)
	}
}

func TestParse_WhitespaceAroundColon(t *testing.T) {
	p := testParser()

	tests := []string{
		"Проект:ТРК",
		"Проект: ТРК",
		"Проект  :  ТРК",
	}

	for _, text := range tests {
		result := p.Parse(text)
		if result.Project != "ТРК" {
			t.Errorf("text %q: Project = %q, want %q", text, result.Project, "ТРК")
		}
	}
}

func TestParse_CaseInsensitiveKeys(t *testing.T) {
	p := testParser()

	tests := []string{
		"проект: ТРК",
		"ПРОЕКТ: ТРК",
		"ПроеКт: ТРК",
		"project: TRK",
		"PROJECT: TRK",
	}

	for _, text := range tests {
		result := p.Parse(text)
		expected := "ТРК"
		if strings.Contains(strings.ToLower(text), "trk") {
			expected = "TRK"
		}
		if result.Project != expected {
			t.Errorf("text %q: Project = %q, want %q", text, result.Project, expected)
		}
	}
}

func TestParse_DuplicateKeys(t *testing.T) {
	p := testParser()

	text := `Проект: ТРК
Проект: Отель`

	result := p.Parse(text)

	if result.Project != "ТРК" {
		t.Errorf("Project = %q, want first value %q", result.Project, "ТРК")
	}
}

func TestParse_OnlyType(t *testing.T) {
	p := testParser()

	text := `Тип: bug
Описание проблемы`

	result := p.Parse(text)

	if !result.HasFields {
		t.Error("expected HasFields=true")
	}
	if result.Type != "bug" {
		t.Errorf("Type = %q", result.Type)
	}
	if result.Project != "" {
		t.Errorf("Project should be empty, got %q", result.Project)
	}
}

func TestParse_BodyPreservedWithFields(t *testing.T) {
	p := testParser()

	text := `Проект: Отель
Тип: feature

Первая строка описания.
Вторая строка описания.
Третья строка.`

	result := p.Parse(text)

	expectedBody := "Первая строка описания.\nВторая строка описания.\nТретья строка."
	if result.Body != expectedBody {
		t.Errorf("Body = %q, want %q", result.Body, expectedBody)
	}
}

func TestParse_AllValidTypes(t *testing.T) {
	p := testParser()

	validTypes := []string{"bug", "feature", "support", "access", "seo", "content"}

	for _, typ := range validTypes {
		result := p.Parse("Тип: " + typ)
		if result.Type != typ {
			t.Errorf("Type %q was not accepted", typ)
		}
	}
}

func TestParse_AllValidPriorities(t *testing.T) {
	p := testParser()

	validPrios := []string{"urgent", "high", "medium", "low"}

	for _, prio := range validPrios {
		result := p.Parse("Приоритет: " + prio)
		expected := normalizeCase(prio)
		if result.Priority != expected {
			t.Errorf("Priority %q -> %q, want %q", prio, result.Priority, expected)
		}
	}
}

// normalizeCase вспомогательная функция для тестов.
func normalizeCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) == 1 {
		return strings.ToUpper(string(runes[0]))
	}
	return strings.ToUpper(string(runes[0])) + strings.ToLower(string(runes[1:]))
}
