package prep

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultQuestionCount = 8

	envPrepEnabled              = "T2O_PREP_ENABLED"
	envPrepDataDir              = "T2O_PREP_DATA_DIR"
	envPrepDefaultQuestionCount = "T2O_PREP_DEFAULT_QUESTION_COUNT"
)

type Config struct {
	Enabled              bool
	DataDir              string
	DefaultQuestionCount int
	SupportedScopes      []Scope
}

func LoadConfig(dataRootDir string) (Config, error) {
	return loadConfig(dataRootDir, os.Getenv)
}

func loadConfig(dataRootDir string, getenv func(string) string) (Config, error) {
	root := strings.TrimSpace(dataRootDir)
	if root == "" {
		return Config{}, fmt.Errorf("prep data root dir is required")
	}

	enabled := true
	if raw := strings.TrimSpace(getenv(envPrepEnabled)); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s=%q: %w", envPrepEnabled, raw, err)
		}
		enabled = value
	}

	questionCount := defaultQuestionCount
	if raw := strings.TrimSpace(getenv(envPrepDefaultQuestionCount)); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s=%q: %w", envPrepDefaultQuestionCount, raw, err)
		}
		if value <= 0 {
			return Config{}, fmt.Errorf("%s must be greater than 0", envPrepDefaultQuestionCount)
		}
		questionCount = value
	}

	dataDir := strings.TrimSpace(getenv(envPrepDataDir))
	if dataDir == "" {
		dataDir = filepath.Join(root, "prep")
	} else if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(root, dataDir)
	}
	dataDir = filepath.Clean(dataDir)

	config := Config{
		Enabled:              enabled,
		DataDir:              dataDir,
		DefaultQuestionCount: questionCount,
		SupportedScopes:      DefaultSupportedScopes(),
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("prep data dir is required")
	}
	if c.DefaultQuestionCount <= 0 {
		return fmt.Errorf("prep default question count must be greater than 0")
	}
	if len(c.SupportedScopes) == 0 {
		return fmt.Errorf("prep supported scopes are required")
	}
	for _, scope := range c.SupportedScopes {
		if !isSupportedScope(scope) {
			return fmt.Errorf("prep scope %q is unsupported", scope)
		}
	}
	return nil
}
