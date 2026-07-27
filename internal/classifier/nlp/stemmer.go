package nlp

import (
	"strings"
	"unicode/utf8"
)

// Stemmer возвращает основу слова.
type Stemmer interface {
	Stem(word string) string
}

// RussianStemmer реализует стеммер Портера для русского языка.
// Алгоритм: https://snowballstem.org/algorithms/russian/stemmer.html
type RussianStemmer struct{}

// NewRussianStemmer создаёт новый RussianStemmer.
func NewRussianStemmer() *RussianStemmer {
	return &RussianStemmer{}
}

// Stem возвращает основу русского слова по алгоритму Портера.
func (s *RussianStemmer) Stem(word string) string {
	word = strings.ToLower(word)
	word = strings.ReplaceAll(word, "ё", "е")

	if utf8.RuneCountInString(word) <= 2 {
		return word
	}

	// markRegions теперь возвращает позиции, а не только строки
	rv, rvStart, r2Start := s.markRegions(word)

	// Шаг 1
	word = s.step1(word, rv)

	// Шаг 2: "и" в RV текущего слова
	word = s.step2(word, rvStart)

	// Шаг 3: DERIVATIONAL — проверяем суффикс на текущем слове,
	// но убеждаемся, что он начинается внутри R2
	word = s.step3(word, r2Start)

	// Шаг 4
	word = s.step4(word)

	return word
}

// markRegions теперь возвращает rv-строку и байтовые позиции rvStart/r2Start
func (s *RussianStemmer) markRegions(word string) (rv string, rvStart, r2Start int) {
	runes := []rune(word)

	// RV: после первой гласной
	rvRuneIdx := -1
	for i, r := range runes {
		if isVowel(r) {
			rvRuneIdx = i + 1
			break
		}
	}
	if rvRuneIdx > 0 {
		rvStart = len(string(runes[:rvRuneIdx]))
		if rvStart < len(word) {
			rv = word[rvStart:]
		}
	}

	// R1: после первого non-vowel, следующего за vowel
	r1RuneIdx := -1
	for i := 0; i < len(runes)-1; i++ {
		if isVowel(runes[i]) && !isVowel(runes[i+1]) {
			r1RuneIdx = i + 2
			break
		}
	}

	// R2: после первого non-vowel, следующего за vowel в R1
	if r1RuneIdx > 0 {
		for i := r1RuneIdx; i < len(runes)-1; i++ {
			if isVowel(runes[i]) && !isVowel(runes[i+1]) {
				r2RuneIdx := i + 2
				r2Start = len(string(runes[:r2RuneIdx]))
				break
			}
		}
	}

	return
}

// step1: PERFECTIVE GERUND, REFLEXIVE, ADJECTIVAL, VERB, NOUN.
func (s *RussianStemmer) step1(word, rv string) string {
	if w := s.removePerfectiveGerund(word, rv); w != word {
		return w
	}

	word, rv = s.removeReflexive(word, rv)

	if w := s.removeAdjectival(word, rv); w != word {
		return w
	}

	if w := s.removeVerb(word, rv); w != word {
		return w
	}

	if w := s.removeNoun(word, rv); w != word {
		return w
	}

	return word
}

func (s *RussianStemmer) removePerfectiveGerund(word, rv string) string {
	group1 := []string{"вшись", "вши", "в"}
	for _, suffix := range group1 {
		if strings.HasSuffix(rv, suffix) {
			stemRV := rv[:len(rv)-len(suffix)]
			if len(stemRV) > 0 {
				lastRune, _ := utf8.DecodeLastRuneInString(stemRV)
				if lastRune == 'а' || lastRune == 'я' {
					return word[:len(word)-len(suffix)]
				}
			}
		}
	}

	group2 := []string{"ившись", "ывшись", "ивши", "ывши", "ив", "ыв"}
	for _, suffix := range group2 {
		if strings.HasSuffix(rv, suffix) {
			return word[:len(word)-len(suffix)]
		}
	}

	return word
}

func (s *RussianStemmer) removeReflexive(word, rv string) (string, string) {
	suffixes := []string{"ся", "сь"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(rv, suffix) {
			return word[:len(word)-len(suffix)], rv[:len(rv)-len(suffix)]
		}
	}
	return word, rv
}

// ADJECTIVAL = ADJECTIVE suffix, опционально предварённый PARTICIPLE suffix.
func (s *RussianStemmer) removeAdjectival(word, rv string) string {
	adjective := []string{
		"ее", "ие", "ые", "ое", "ими", "ыми", "ей", "ий", "ый", "ой",
		"ем", "им", "ым", "ом", "его", "ого", "ему", "ому", "их", "ых",
		"ую", "юю", "ая", "яя", "ою", "ею",
	}

	bestAdj := ""
	for _, adj := range adjective {
		if strings.HasSuffix(rv, adj) && len(adj) > len(bestAdj) {
			bestAdj = adj
		}
	}
	if bestAdj == "" {
		return word
	}

	newWord := word[:len(word)-len(bestAdj)]
	newRV := rv[:len(rv)-len(bestAdj)]

	participle1 := []string{"ем", "нн", "вш", "ющ", "щ"}
	participle2 := []string{"ивш", "ывш", "ующ"}

	bestPart := ""
	isGroup1 := false

	for _, part := range participle1 {
		if strings.HasSuffix(newRV, part) && len(part) > len(bestPart) {
			bestPart = part
			isGroup1 = true
		}
	}
	for _, part := range participle2 {
		if strings.HasSuffix(newRV, part) && len(part) > len(bestPart) {
			bestPart = part
			isGroup1 = false
		}
	}

	if bestPart != "" {
		if isGroup1 {
			stemRV := newRV[:len(newRV)-len(bestPart)]
			if len(stemRV) > 0 {
				lastRune, _ := utf8.DecodeLastRuneInString(stemRV)
				if lastRune == 'а' || lastRune == 'я' {
					return newWord[:len(newWord)-len(bestPart)]
				}
			}
		} else {
			return newWord[:len(newWord)-len(bestPart)]
		}
	}

	return newWord
}

func (s *RussianStemmer) removeVerb(word, rv string) string {
	group1 := []string{
		"ла", "на", "ете", "йте", "ли", "й", "л", "ем", "н", "ло", "но",
		"ет", "ют", "ны", "ть", "ешь", "нно",
	}
	group2 := []string{
		"ила", "ыла", "ена", "ейте", "уйте", "ите", "или", "ыли", "ей",
		"уй", "ил", "ыл", "им", "ым", "ен", "ило", "ыло", "ено", "ят",
		"ует", "уют", "ит", "ыт", "ены", "ить", "ыть", "ишь", "ую", "ю",
	}

	bestSuffix := ""

	for _, suffix := range group1 {
		if strings.HasSuffix(rv, suffix) && len(suffix) > len(bestSuffix) {
			stemRV := rv[:len(rv)-len(suffix)]
			if len(stemRV) > 0 {
				lastRune, _ := utf8.DecodeLastRuneInString(stemRV)
				if lastRune == 'а' || lastRune == 'я' {
					bestSuffix = suffix
				}
			}
		}
	}

	for _, suffix := range group2 {
		if strings.HasSuffix(rv, suffix) && len(suffix) > len(bestSuffix) {
			bestSuffix = suffix
		}
	}

	if bestSuffix != "" {
		return word[:len(word)-len(bestSuffix)]
	}
	return word
}

func (s *RussianStemmer) removeNoun(word, rv string) string {
	suffixes := []string{
		"а", "ев", "ов", "ие", "ье", "е", "иями", "ями", "ами", "еи", "ии",
		"и", "ией", "ей", "ой", "ий", "й", "иям", "ям", "ием", "ем", "ам",
		"ом", "о", "у", "ах", "иях", "ях", "ы", "ь", "ию", "ью", "ю", "ия",
		"ья", "я",
	}

	bestSuffix := ""
	for _, suffix := range suffixes {
		if strings.HasSuffix(rv, suffix) && len(suffix) > len(bestSuffix) {
			bestSuffix = suffix
		}
	}

	if bestSuffix != "" {
		return word[:len(word)-len(bestSuffix)]
	}
	return word
}

// step2 теперь использует rvStart (байтовая позиция), а не строку rv
func (s *RussianStemmer) step2(word string, rvStart int) string {
	if strings.HasSuffix(word, "и") {
		suffixStart := len(word) - len("и")
		if suffixStart >= rvStart {
			return word[:suffixStart]
		}
	}
	return word
}

// step3: ищем суффикс на конце ТЕКУЩЕГО слова, но проверяем, что он внутри R2
func (s *RussianStemmer) step3(word string, r2Start int) string {
	if r2Start <= 0 {
		return word
	}

	derivational := []string{"ость", "ост"}
	bestSuffix := ""
	for _, suffix := range derivational {
		if strings.HasSuffix(word, suffix) && len(suffix) > len(bestSuffix) {
			suffixStart := len(word) - len(suffix)
			if suffixStart >= r2Start {
				bestSuffix = suffix
			}
		}
	}
	if bestSuffix != "" {
		return word[:len(word)-len(bestSuffix)]
	}
	return word
}

func (s *RussianStemmer) step4(word string) string {
	if strings.HasSuffix(word, "нн") {
		return word[:len(word)-len("н")]
	}

	superlative := []string{"ейше", "ейш"}
	for _, suffix := range superlative {
		if strings.HasSuffix(word, suffix) {
			word = word[:len(word)-len(suffix)]
			if strings.HasSuffix(word, "нн") {
				word = word[:len(word)-len("н")]
			}
			return word
		}
	}

	if strings.HasSuffix(word, "ь") {
		return word[:len(word)-len("ь")]
	}

	return word
}

func isVowel(r rune) bool {
	vowels := "аеиоуыэюя"
	return strings.ContainsRune(vowels, r)
}

// EnglishStemmer — упрощённый стеммер для английского.
type EnglishStemmer struct{}

func NewEnglishStemmer() *EnglishStemmer {
	return &EnglishStemmer{}
}

func (s *EnglishStemmer) Stem(word string) string {
	word = strings.ToLower(word)

	suffixes := []string{"ing", "ed", "es", "s", "ly", "ment", "ness", "tion", "er", "est"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word)-len(suffix) >= 3 {
			word = word[:len(word)-len(suffix)]
			break
		}
	}

	if len(word) >= 3 && word[len(word)-1] == word[len(word)-2] {
		word = word[:len(word)-1]
	}

	return word
}
