package prep

type EmbeddingProvider interface {
	Embed(texts []string) ([][]float32, error)
}

type EmbeddingProviderValidator interface {
	Validate() error
}

type EmbeddingProviderInfo interface {
	Name() string
	Model() string
	Dimension() int
}
