package core

import (
	"regexp"
	"strings"
)

type PrivacyFilter struct {
	// If true, patterns are treated as regex. If false, simple substring match.
	UseRegex bool

	// Patterns to ignore (e.g. "token=", "password=", "Authorization: Bearer")
	Patterns []string

	compiled []*regexp.Regexp
}

func NewPrivacyFilter(patterns []string, useRegex bool) (*PrivacyFilter, error) {
	pf := &PrivacyFilter{
		UseRegex: useRegex,
		Patterns: patterns,
	}
	if useRegex {
		pf.compiled = make([]*regexp.Regexp, 0, len(patterns))
		for _, p := range patterns {
			if strings.TrimSpace(p) == "" {
				continue
			}
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, err
			}
			pf.compiled = append(pf.compiled, re)
		}
	}
	return pf, nil
}

func (pf *PrivacyFilter) ShouldIgnore(content string) bool {
	if pf == nil {
		return false
	}
	s := strings.TrimSpace(content)
	if s == "" {
		return true // ignore empty
	}

	if pf.UseRegex {
		for _, re := range pf.compiled {
			if re.MatchString(s) {
				return true
			}
		}
		return false
	}

	low := strings.ToLower(s)
	for _, p := range pf.Patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
