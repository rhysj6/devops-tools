package pfp

import (
	"regexp"
	"strings"
)

type LineMatcher struct {
	Contains  string
	RegexText string         `mapstructure:"regex"`
	Regex     *regexp.Regexp `mapstructure:"-"`
}

type Rule struct {
	Name     string
	Checks   []LineMatcher `mapstructure:"patterns"`
	MaxLines int           `mapstructure:"maxlines"`
	Solution string
}

func (r Rule) GetNeededLineCount() int {
	if len(r.Checks) > r.MaxLines {
		return len(r.Checks)
	}

	if len(r.Checks) == 1 {
		return 1
	}

	return r.MaxLines
}

func (m *LineMatcher) CheckLine(l string) bool {
	hasRegex := m.Regex != nil

	if m.Contains != "" {
		if !strings.Contains(l, m.Contains) {
			return false
		} else if !hasRegex {
			return true
		}
	}
	if !hasRegex {
		return false
	}

	return m.Regex.FindString(l) != ""
}
