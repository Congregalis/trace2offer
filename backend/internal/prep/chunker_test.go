package prep

import (
	"strings"
	"testing"
)

func TestNewChunkerDefaults(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(ChunkConfig{})
	if err != nil {
		t.Fatalf("new chunker: %v", err)
	}

	config := chunker.Config()
	if config.ChunkSize != defaultChunkSizeTokens {
		t.Fatalf("expected default chunk size=%d, got %d", defaultChunkSizeTokens, config.ChunkSizeTokens)
	}
	if config.Overlap != defaultChunkOverlap {
		t.Fatalf("expected default overlap=%d, got %d", defaultChunkOverlap, config.OverlapTokens)
	}
}

func TestChunkerSplitWithOverlap(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(ChunkConfig{ChunkSize: 4, Overlap: 1})
	if err != nil {
		t.Fatalf("new chunker: %v", err)
	}

	content := strings.Repeat("a", 40)
	chunks := chunker.Chunk(content, ChunkConfig{ChunkSize: 4, Overlap: 1})
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0].Content) != 16 || len(chunks[1].Content) != 16 || len(chunks[2].Content) != 16 {
		t.Fatalf("unexpected chunk sizes: %+v", chunks)
	}
	if chunks[0].Index != 0 || chunks[1].Index != 1 || chunks[2].Index != 2 {
		t.Fatalf("unexpected chunk indexes: %+v", chunks)
	}
	if chunks[0].TokenCount != 4 {
		t.Fatalf("expected first chunk token_count=4, got %d", chunks[0].TokenCount)
	}
}

func TestChunkerSplitEmptyContent(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(ChunkConfig{ChunkSize: 16, Overlap: 2})
	if err != nil {
		t.Fatalf("new chunker: %v", err)
	}
	chunks := chunker.Chunk("   \n\t", ChunkConfig{})
	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for whitespace content, got %d", len(chunks))
	}
}

func TestNewChunkerOverlapNormalization(t *testing.T) {
	t.Parallel()

	chunker, err := NewChunker(ChunkConfig{ChunkSize: 8, Overlap: 8})
	if err != nil {
		t.Fatalf("new chunker: %v", err)
	}
	config := chunker.Config()
	if config.Overlap != 7 {
		t.Fatalf("expected normalized overlap=7, got %d", config.Overlap)
	}
}
