package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

// LoadDotEnv loads key/value pairs from a dotenv file into process env.
// Existing environment variables are kept as-is.
func LoadDotEnv(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("dotenv path is required")
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open dotenv file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid dotenv line %d: missing '='", lineNo)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("invalid dotenv line %d: empty key", lineNo)
		}

		value = normalizeValue(strings.TrimSpace(value))
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read dotenv file: %w", err)
	}
	return nil
}

func normalizeValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if (strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`)) ||
		(strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'")) {
		return raw[1 : len(raw)-1]
	}

	if commentAt := strings.Index(raw, " #"); commentAt >= 0 {
		return strings.TrimSpace(raw[:commentAt])
	}

	return raw
}
