// Package nlp предоставляет инструменты нормализации текста:
// токенизацию, стемминг и генерацию n-грамм.
package nlp

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Tokenizer разбивает текст на токены, удаляя стоп-слова и знаки препинания.
type Tokenizer struct {
	stopWords map[string]bool
}

// NewTokenizer создаёт новый Tokenizer с дефолтным набором стоп-слов.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		stopWords: defaultStopWords(),
	}
}

// NewTokenizerWithStopWords создаёт Tokenizer с пользовательским набором стоп-слов.
func NewTokenizerWithStopWords(stopWords []string) *Tokenizer {
	sw := make(map[string]bool, len(stopWords))
	for _, w := range stopWords {
		sw[strings.ToLower(w)] = true
	}
	return &Tokenizer{stopWords: sw}
}

// Tokenize разбивает текст на значимые токены.
func (t *Tokenizer) Tokenize(text string) []string {
	// Приводим к нижнему регистру
	text = strings.ToLower(text)

	// Удаляем все символы, кроме букв и пробелов (сохраняем дефисы для составных слов)
	reg := regexp.MustCompile(`[^\p{L}\s-]+`)
	text = reg.ReplaceAllString(text, " ")

	// Разбиваем на слова
	rawWords := strings.Fields(text)

	// Фильтруем стоп-слова и короткие слова
	var tokens []string
	for _, w := range rawWords {
		w = strings.TrimSpace(w)
		// Пропускаем слова короче 2 символов
		if utf8.RuneCountInString(w) <= 1 {
			continue
		}
		// Пропускаем стоп-слова
		if t.stopWords[w] {
			continue
		}
		// Пропускаем слова, состоящие только из дефисов
		if strings.Trim(w, "-") == "" {
			continue
		}
		tokens = append(tokens, w)
	}

	return tokens
}

// defaultStopWords возвращает стандартный набор стоп-слов для русского и английского.
func defaultStopWords() map[string]bool {
	words := []string{
		// Русские
		"в", "на", "с", "и", "а", "но", "что", "это", "как", "для",
		"из", "от", "по", "к", "у", "о", "не", "за", "то", "мы",
		"я", "он", "она", "они", "вы", "ты", "бы", "ли", "же",
		"уже", "там", "тут", "где", "когда", "почему", "который",
		"весь", "очень", "так", "более", "менее", "ещё", "всё",
		"есть", "быть", "был", "была", "были", "будет", "будут",
		"можно", "нужно", "надо", "нет", "да", "или", "чтобы",
		"под", "над", "при", "без", "до", "после", "во", "со",
		// Вежливые обращения
		"добрый", "день", "здравствуйте", "привет", "спасибо",
		"пожалуйста", "будьте", "добры", "прошу", "утро", "вечер",
		// English
		"the", "a", "an", "is", "are", "was", "were", "be", "been",
		"being", "have", "has", "had", "do", "does", "did", "will",
		"would", "could", "should", "may", "might", "can", "shall",
		"i", "you", "he", "she", "it", "we", "they", "me", "him",
		"her", "us", "them", "my", "your", "his", "its", "our",
		"to", "of", "in", "for", "on", "with", "at", "by", "from",
		"and", "or", "but", "not", "so", "if", "then", "than",
		"too", "very", "just", "about", "also", "now", "please",
		"hello", "hi", "thanks", "thank",
	}

	sw := make(map[string]bool, len(words))
	for _, w := range words {
		sw[w] = true
	}
	return sw
}
