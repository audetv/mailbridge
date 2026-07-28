package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RulesConfig содержит конфигурацию правил классификации.
type RulesConfig struct {
	Rules           []RuleDef `yaml:"rules"`
	UrgencyPatterns []string  `yaml:"urgency_patterns"`
	ValidTypes      []string  `yaml:"valid_types"`
	ValidPriorities []string  `yaml:"valid_priorities"`
}

// RuleDef описывает одно правило из YAML.
type RuleDef struct {
	Keywords []string `yaml:"keywords"`
	Project  string   `yaml:"project,omitempty"`
	Type     string   `yaml:"type,omitempty"`
	Priority string   `yaml:"priority,omitempty"`
	Weight   int      `yaml:"weight"`
}

// LoadRules загружает правила из YAML-файла.
func LoadRules(path string) (*RulesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", path, err)
	}

	var cfg RulesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse rules file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid rules config: %w", err)
	}

	return &cfg, nil
}

// validate проверяет обязательные поля конфигурации правил.
func (c *RulesConfig) validate() error {
	if len(c.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}
	for i, r := range c.Rules {
		if len(r.Keywords) == 0 {
			return fmt.Errorf("rule %d: keywords are required", i)
		}
		if r.Weight <= 0 {
			return fmt.Errorf("rule %d: weight must be positive", i)
		}
	}
	if len(c.UrgencyPatterns) == 0 {
		return fmt.Errorf("at least one urgency pattern is required")
	}
	if len(c.ValidTypes) == 0 {
		return fmt.Errorf("valid_types is required")
	}
	if len(c.ValidPriorities) == 0 {
		return fmt.Errorf("valid_priorities is required")
	}
	return nil
}

// ToSliceMap возвращает validTypes как map для быстрого поиска.
func (c *RulesConfig) ValidTypesMap() map[string]bool {
	m := make(map[string]bool, len(c.ValidTypes))
	for _, t := range c.ValidTypes {
		m[t] = true
	}
	return m
}

// ValidPrioritiesMap возвращает validPriorities как map.
func (c *RulesConfig) ValidPrioritiesMap() map[string]bool {
	m := make(map[string]bool, len(c.ValidPriorities))
	for _, p := range c.ValidPriorities {
		m[p] = true
	}
	return m
}
