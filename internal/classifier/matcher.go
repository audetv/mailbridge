package classifier

import (
	"sort"
	"strings"

	"github.com/audetv/mailbridge/internal/classifier/nlp"
)

// ScoredMatch содержит результат сопоставления правила с текстом.
type ScoredMatch struct {
	Rule    Rule
	Score   int
	Matches []string
}

// Matcher сопоставляет текст с правилами, используя NLP-нормализацию.
type Matcher struct {
	tokenizer       *nlp.Tokenizer
	stemmer         nlp.Stemmer
	ngramGen        *nlp.NGramGenerator
	rules           []Rule
	normalizedRules map[int][]string
}

// NewMatcher создаёт новый Matcher с русским стеммером.
func NewMatcher(rules []Rule) *Matcher {
	m := &Matcher{
		tokenizer:       nlp.NewTokenizer(),
		stemmer:         nlp.NewRussianStemmer(),
		ngramGen:        nlp.NewNGramGenerator(),
		rules:           rules,
		normalizedRules: make(map[int][]string),
	}

	for i, rule := range rules {
		var allNGrams []string
		for _, keyword := range rule.Keywords {
			phraseNGrams := m.ngramGen.GenerateFromPhrase(keyword, m.tokenizer, m.stemmer)
			allNGrams = append(allNGrams, phraseNGrams...)
		}
		m.normalizedRules[i] = uniqueStrings(allNGrams)
	}

	return m
}

// Match прогоняет текст через все правила и возвращает отсортированные совпадения.
func (m *Matcher) Match(text string) []ScoredMatch {
	tokens := m.tokenizer.Tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	stemmed := make([]string, len(tokens))
	for i, token := range tokens {
		stemmed[i] = m.stemmer.Stem(token)
	}

	textNGrams := m.ngramGen.Generate(stemmed)
	textNGramSet := toSet(textNGrams)

	var matches []ScoredMatch

	for ruleIdx, ruleNGrams := range m.normalizedRules {
		var matchedNGrams []string
		score := 0

		for _, ruleNGram := range ruleNGrams {
			if textNGramSet[ruleNGram] {
				matchedNGrams = append(matchedNGrams, ruleNGram)

				underscores := strings.Count(ruleNGram, "_")
				switch underscores {
				case 0:
					score++
				case 1:
					score += 3
				case 2:
					score += 5
				}
			}
		}

		if score > 0 {
			score *= m.rules[ruleIdx].Weight
			matches = append(matches, ScoredMatch{
				Rule:    m.rules[ruleIdx],
				Score:   score,
				Matches: matchedNGrams,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
