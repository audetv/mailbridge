package nlp_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/audetv/mailbridge/internal/classifier/nlp"
)

func TestTokenizer_Basic(t *testing.T) {
	tokenizer := nlp.NewTokenizer()

	tokens := tokenizer.Tokenize("Не работает сайт ТРК")
	if len(tokens) == 0 {
		t.Fatal("expected tokens, got none")
	}

	for _, token := range tokens {
		if token == "не" {
			t.Error("'не' should be removed as stop word")
		}
	}

	expected := []string{"работает", "сайт", "трк"}
	for _, exp := range expected {
		found := false
		for _, tok := range tokens {
			if tok == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected token %q not found in %v", exp, tokens)
		}
	}
}

func TestTokenizer_StopWords(t *testing.T) {
	tokenizer := nlp.NewTokenizer()

	tests := []struct {
		input    string
		expected []string
	}{
		{"добрый день помогите с доступом", []string{"помогите", "доступом"}},
		{"здравствуйте нужен пароль", []string{"нужен", "пароль"}},
		{"пожалуйста обновите баннер", []string{"обновите", "баннер"}},
	}

	for _, tt := range tests {
		tokens := tokenizer.Tokenize(tt.input)
		sort.Strings(tokens)
		sort.Strings(tt.expected)
		if !reflect.DeepEqual(tokens, tt.expected) {
			t.Errorf("Tokenize(%q) = %v, want %v", tt.input, tokens, tt.expected)
		}
	}
}

func TestTokenizer_Punctuation(t *testing.T) {
	tokenizer := nlp.NewTokenizer()

	tokens := tokenizer.Tokenize("Ошибка 500! Сайт упал!!!")
	found500 := false
	for _, tok := range tokens {
		if tok == "500" {
			found500 = true
		}
	}
	if found500 {
		t.Error("numbers should be removed")
	}
}

func TestTokenizer_CustomStopWords(t *testing.T) {
	custom := []string{"трк", "отель"}
	tokenizer := nlp.NewTokenizerWithStopWords(custom)

	tokens := tokenizer.Tokenize("сайт трк не работает")
	for _, tok := range tokens {
		if tok == "трк" {
			t.Error("'трк' should be removed as custom stop word")
		}
	}
}

func TestTokenizer_ShortWords(t *testing.T) {
	tokenizer := nlp.NewTokenizer()

	tokens := tokenizer.Tokenize("я и ты")
	if len(tokens) != 0 {
		t.Errorf("expected no tokens, got %v", tokens)
	}
}

// TestRussianStemmer проверяет реализацию алгоритма Портера для русского языка.
// Ожидаемые результаты соответствуют официальному алгоритму Snowball:
// https://snowballstem.org/algorithms/russian/stemmer.html
func TestRussianStemmer(t *testing.T) {
	stemmer := nlp.NewRussianStemmer()

	tests := []struct {
		input    string
		expected string
	}{
		// Глаголы
		{"работает", "работа"},
		{"работающий", "работа"},   // ADJECTIVAL: ющ + ий
		{"работала", "работа"},     // VERB group1: ла
		{"открывается", "открыва"}, // REFLEXIVE + VERB
		{"открыть", "откр"},        // VERB group2: ыть (инфинитив)

		// Существительные
		{"ошибка", "ошибк"},
		{"ошибки", "ошибк"},
		{"сервера", "сервер"},
		{"пользователей", "пользовател"},
		{"доступ", "доступ"},
		{"обновление", "обновлен"},

		// Прилагательные (ADJECTIVAL) — по Портеру остаётся "н"
		{"ошибочный", "ошибочн"},

		// Совершенный вид деепричастия (PERFECTIVE GERUND)
		{"прочитав", "прочита"}, // group1: в после а
		{"сделавши", "сдела"},   // group1: вши после а

		// Возвратные глаголы
		{"смотрится", "смотр"}, // REFLEXIVE + VERB group2: ит

		// Причастия + прилагательные
		{"бегавшая", "бега"}, // PARTICIPLE вш + ADJECTIVE ая

		// Деривational (R2)
		{"безопасность", "безопасн"},

		// Превосходная степень
		{"сильнейший", "сильн"},

		// Мягкий знак
		{"пароль", "парол"},
	}

	for _, tt := range tests {
		result := stemmer.Stem(tt.input)
		if result != tt.expected {
			t.Errorf("Stem(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRussianStemmer_ShortWords(t *testing.T) {
	stemmer := nlp.NewRussianStemmer()

	// Слова ≤ 2 руны не обрабатываются
	if stemmer.Stem("он") != "он" {
		t.Error("short word should not be stemmed")
	}
}

func TestEnglishStemmer(t *testing.T) {
	stemmer := nlp.NewEnglishStemmer()

	tests := []struct {
		input    string
		expected string
	}{
		{"running", "run"},
		{"working", "work"},
		{"errors", "error"},
		{"connection", "connec"},
		{"happiness", "happines"},
		{"easily", "easi"},
	}

	for _, tt := range tests {
		result := stemmer.Stem(tt.input)
		if result != tt.expected {
			t.Errorf("Stem(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNGramGenerator_Unigrams(t *testing.T) {
	gen := nlp.NewNGramGenerator()

	tokens := []string{"ошибк", "сервер", "доступ"}
	ngrams := gen.Generate(tokens)

	for _, token := range tokens {
		found := false
		for _, ng := range ngrams {
			if ng == token {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unigram %q not found", token)
		}
	}
}

func TestNGramGenerator_Bigrams(t *testing.T) {
	gen := nlp.NewNGramGenerator()

	tokens := []string{"ошибк", "сервер", "доступ"}
	ngrams := gen.Generate(tokens)

	expectedBigrams := []string{"ошибк_сервер", "сервер_доступ"}
	for _, expected := range expectedBigrams {
		found := false
		for _, ng := range ngrams {
			if ng == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("bigram %q not found in %v", expected, ngrams)
		}
	}
}

func TestNGramGenerator_Trigrams(t *testing.T) {
	gen := nlp.NewNGramGenerator()

	tokens := []string{"не", "работает", "сайт"}
	ngrams := gen.Generate(tokens)

	expected := "не_работает_сайт"
	found := false
	for _, ng := range ngrams {
		if ng == expected {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("trigram %q not found in %v", expected, ngrams)
	}
}

func TestNGramGenerator_SingleToken(t *testing.T) {
	gen := nlp.NewNGramGenerator()

	tokens := []string{"ошибка"}
	ngrams := gen.Generate(tokens)

	if len(ngrams) != 1 {
		t.Errorf("expected 1 ngram, got %d: %v", len(ngrams), ngrams)
	}
}

func TestNGramGenerator_Empty(t *testing.T) {
	gen := nlp.NewNGramGenerator()

	ngrams := gen.Generate(nil)
	if ngrams != nil {
		t.Error("expected nil for empty input")
	}

	ngrams = gen.Generate([]string{})
	if ngrams != nil {
		t.Error("expected nil for empty slice")
	}
}

func TestGenerateFromPhrase(t *testing.T) {
	gen := nlp.NewNGramGenerator()
	tokenizer := nlp.NewTokenizer()
	stemmer := nlp.NewRussianStemmer()

	ngrams := gen.GenerateFromPhrase("не работает сервер", tokenizer, stemmer)

	if len(ngrams) < 2 {
		t.Fatalf("expected at least 2 ngrams, got %d: %v", len(ngrams), ngrams)
	}

	foundBigram := false
	for _, ng := range ngrams {
		if ng == "работа_сервер" || ng == "сервер_работа" {
			foundBigram = true
		}
	}
	if !foundBigram {
		t.Error("bigram not found")
	}
}
