package nlp

// NGramGenerator генерирует n-граммы из токенов.
type NGramGenerator struct{}

// NewNGramGenerator создаёт новый NGramGenerator.
func NewNGramGenerator() *NGramGenerator {
	return &NGramGenerator{}
}

// Generate возвращает униграммы, биграммы и триграммы из списка токенов.
func (g *NGramGenerator) Generate(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}

	// Оценка ёмкости: n униграмм + (n-1) биграмм + (n-2) триграмм
	capacity := len(tokens)
	if len(tokens) > 1 {
		capacity += len(tokens) - 1
	}
	if len(tokens) > 2 {
		capacity += len(tokens) - 2
	}

	ngrams := make([]string, 0, capacity)

	// Униграммы
	ngrams = append(ngrams, tokens...)

	// Биграммы
	for i := 0; i < len(tokens)-1; i++ {
		ngrams = append(ngrams, tokens[i]+"_"+tokens[i+1])
	}

	// Триграммы
	for i := 0; i < len(tokens)-2; i++ {
		ngrams = append(ngrams, tokens[i]+"_"+tokens[i+1]+"_"+tokens[i+2])
	}

	return ngrams
}

// GenerateFromPhrase токенизирует фразу, стеммит и генерирует n-граммы.
// Используется для подготовки ключевых фраз из правил.
func (g *NGramGenerator) GenerateFromPhrase(phrase string, tokenizer *Tokenizer, stemmer Stemmer) []string {
	tokens := tokenizer.Tokenize(phrase)
	if len(tokens) == 0 {
		return nil
	}

	stemmed := make([]string, len(tokens))
	for i, t := range tokens {
		stemmed[i] = stemmer.Stem(t)
	}

	return g.Generate(stemmed)
}
