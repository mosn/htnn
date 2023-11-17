package expr

import (
	"regexp"
	"strings"

	api "mosn.io/moe/api/v1"
)

type Matcher interface {
	Match(s string) bool
	IgnoreCase() bool
}

type stringPrefixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringPrefixMatcher) Match(s string) bool {
	return strings.HasPrefix(s, m.target)
}

func (m *stringPrefixMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringSuffixMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringSuffixMatcher) Match(s string) bool {
	return strings.HasSuffix(s, m.target)
}

func (m *stringSuffixMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringRegexMatcher struct {
	regex *regexp.Regexp
}

func (m *stringRegexMatcher) Match(s string) bool {
	return m.regex.MatchString(s)
}

func (m *stringRegexMatcher) IgnoreCase() bool {
	return false
}

type stringContainsMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringContainsMatcher) Match(s string) bool {
	return strings.Contains(s, m.target)
}

func (m *stringContainsMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type stringExactMatcher struct {
	target     string
	ignoreCase bool
}

func (m *stringExactMatcher) Match(s string) bool {
	return s == m.target
}

func (m *stringExactMatcher) IgnoreCase() bool {
	return m.ignoreCase
}

type repeatedStringMatcher struct {
	needIgnoreCase bool

	matchers []Matcher
}

func (rsm *repeatedStringMatcher) Match(s string) bool {
	var ls string
	if rsm.needIgnoreCase {
		// the repeated string matcher will share one case-insensitive input
		ls = strings.ToLower(s)
	}
	for _, m := range rsm.matchers {
		input := s
		if m.IgnoreCase() {
			input = ls
		}
		if m.Match(input) {
			return true
		}
	}
	return false
}

func (rsm *repeatedStringMatcher) IgnoreCase() bool {
	return rsm.needIgnoreCase
}

func buildRepeatedStringMatcher(matchers []*api.StringMatcher, allIgnoreCase bool) (Matcher, error) {
	builtMatchers := make([]Matcher, len(matchers))
	needIgnoreCase := allIgnoreCase

	// For small input (len(matchers) <= 8), match one by one is faster than creating a match
	// table. Current user case doesn't need to handle big input yet.
	for i, m := range matchers {
		ignoreCase := allIgnoreCase
		if m.IgnoreCase {
			ignoreCase = true
		}

		var matcher Matcher
		switch v := m.MatchPattern.(type) {
		case *api.StringMatcher_Exact:
			target := v.Exact
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringExactMatcher{target: target, ignoreCase: ignoreCase}
		case *api.StringMatcher_Prefix:
			target := v.Prefix
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringPrefixMatcher{target: target, ignoreCase: ignoreCase}
		case *api.StringMatcher_Suffix:
			target := v.Suffix
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringSuffixMatcher{target: target, ignoreCase: ignoreCase}
		case *api.StringMatcher_Contains:
			target := v.Contains
			if ignoreCase {
				target = strings.ToLower(target)
			}
			matcher = &stringContainsMatcher{target: target, ignoreCase: ignoreCase}
		case *api.StringMatcher_Regex:
			target := v.Regex
			if ignoreCase && !strings.HasPrefix(target, "(?i)") {
				target = "(?i)" + target
			}
			re, err := regexp.Compile(target)
			if err != nil {
				return nil, err
			}
			matcher = &stringRegexMatcher{regex: re}
		}

		builtMatchers[i] = matcher

		if ignoreCase {
			needIgnoreCase = true
		}
	}

	return &repeatedStringMatcher{
		matchers:       builtMatchers,
		needIgnoreCase: needIgnoreCase,
	}, nil
}

func BuildRepeatedStringMatcherIgnoreCase(matchers []*api.StringMatcher) (Matcher, error) {
	return buildRepeatedStringMatcher(matchers, true)
}

func BuildRepeatedStringMatcher(matchers []*api.StringMatcher) (Matcher, error) {
	return buildRepeatedStringMatcher(matchers, false)
}

func BuildStringMatcher(m *api.StringMatcher) (Matcher, error) {
	return BuildRepeatedStringMatcher([]*api.StringMatcher{m})
}
