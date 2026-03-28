package prep

import (
	"strings"
)

const (
	defaultChunkSizeTokens = 512
	defaultChunkOverlap    = 50
	tokenApproxDivisor     = 4
)

type TextChunk struct {
	Index      int    `json:"index"`
	StartChar  int    `json:"start_char"`
	EndChar    int    `json:"end_char"`
	Content    string `json:"content"`
	TokenCount int    `json:"token_count"`
}

type Chunker struct {
	defaultConfig ChunkConfig
}

func NewChunker(config ChunkConfig) (*Chunker, error) {
	return &Chunker{defaultConfig: normalizeChunkConfig(config)}, nil
}

func (c *Chunker) Config() ChunkConfig {
	if c == nil {
		return normalizeChunkConfig(ChunkConfig{})
	}
	return normalizeChunkConfig(c.defaultConfig)
}

func (c *Chunker) Chunk(text string, config ChunkConfig) []Chunk {
	effective := normalizeChunkConfig(config)
	if c != nil && effective.ChunkSize <= 0 {
		effective = c.Config()
	}
	return ChunkText(text, effective)
}

func (c *Chunker) Split(content string) []TextChunk {
	chunks := c.Chunk(content, ChunkConfig{})
	out := make([]TextChunk, 0, len(chunks))
	for _, chunk := range chunks {
		out = append(out, TextChunk{
			Index:      chunk.Index,
			Content:    chunk.Content,
			TokenCount: chunk.TokenCount,
		})
	}
	return out
}

func ChunkText(text string, config ChunkConfig) []Chunk {
	if strings.TrimSpace(text) == "" {
		return []Chunk{}
	}

	normalized := normalizeChunkConfig(config)
	runes := []rune(text)
	if len(runes) == 0 {
		return []Chunk{}
	}

	chunkSizeChars := max(1, normalized.ChunkSize*tokenApproxDivisor)
	overlapChars := max(0, normalized.Overlap*tokenApproxDivisor)
	if overlapChars >= chunkSizeChars {
		overlapChars = chunkSizeChars - 1
	}

	chunks := make([]Chunk, 0, 1+len(runes)/chunkSizeChars)
	step := max(1, chunkSizeChars-overlapChars)
	for start := 0; start < len(runes); start += step {
		end := min(start+chunkSizeChars, len(runes))
		chunks = append(chunks, Chunk{
			Index:      len(chunks),
			ChunkIndex: len(chunks),
			Content:    string(runes[start:end]),
			TokenCount: approxTokenCount(end - start),
		})
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func normalizeChunkConfig(config ChunkConfig) ChunkConfig {
	chunkSize := firstPositive(config.ChunkSize, config.ChunkSizeTokens)
	if chunkSize <= 0 {
		chunkSize = defaultChunkSizeTokens
	}
	overlap := firstPositive(config.Overlap, config.OverlapTokens)
	if overlap <= 0 {
		overlap = defaultChunkOverlap
	}
	if overlap >= chunkSize {
		overlap = chunkSize - 1
	}

	return ChunkConfig{
		ChunkSize:       chunkSize,
		Overlap:         overlap,
		ChunkSizeTokens: chunkSize,
		OverlapTokens:   overlap,
	}
}

func approxTokenCount(charCount int) int {
	if charCount <= 0 {
		return 0
	}
	return (charCount + tokenApproxDivisor - 1) / tokenApproxDivisor
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
