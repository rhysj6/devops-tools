package logparser

import (
	"regexp"
	"strings"
)

// LineCheck defines a single line condition using contains and/or regex.
type LineCheck struct {
	// Contains is a simple substring that must be present in the line for it to match. Optional.
	Contains string `json:"contains"`

	// RegexText is a regular expression pattern that the line must match. Optional. If both Contains and RegexText are provided, then both conditions must be satisfied for the line to match.
	RegexText string `mapstructure:"regex" json:"regex"`
	// Regex is the compiled regular expression. This is not set directly in the config, but is populated when the config is loaded and the regex patterns are compiled.
	Regex *regexp.Regexp `mapstructure:"-" json:"-"`

	// MaxLines is the maximum number of lines since the last check was satisfied that this check should be considered a match.
	// E.g. if MaxLines is 5, and this check is the second check in a rule, then this check will be considered satisfied if it matches a line that is within 5 lines of the last line that satisfied the first check in the rule.
	// Optional. If not set, then there is no limit on the number of lines between checks.
	MaxLines int `mapstructure:"maxlines" json:"maxLines"`
}

// MatchRule defines an ordered set of checks and metadata for a log match.
type MatchRule struct {
	Name     string      `json:"name"`
	Checks   []LineCheck `mapstructure:"patterns" json:"patterns"`
	MaxLines int         `mapstructure:"maxlines" json:"maxLines"`
	Solution string      `mapstructure:"solution" json:"solution"`
	Category string      `mapstructure:"category" json:"category"`
}

func (r MatchRule) getNeededLineCount() int {
	if len(r.Checks) <= 1 {
		return 1
	}
	return max(len(r.Checks), r.MaxLines)
}

func (lc *LineCheck) CheckLine(l string) bool {
	if lc.Contains != "" && !strings.Contains(l, lc.Contains) {
		return false
	}
	if lc.Regex != nil {
		return lc.Regex.MatchString(l)
	}
	return lc.Contains != ""
}
