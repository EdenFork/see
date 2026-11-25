package app

import (
	"path/filepath"
	"regexp"
	"strings"
)

type hiddenNameRules struct {
	names, pattern string
	exact          map[string]bool
	regexps        []*regexp.Regexp
	invalidLine    int
}

func (a *App) hiddenRules() *hiddenNameRules {
	if a.cfg == nil {
		return nil
	}
	names, pattern := a.cfg.HiddenNames, a.cfg.HiddenNamePattern
	if rules := a.hidden; rules != nil && rules.names == names && rules.pattern == pattern {
		return rules
	}
	a.hidden = compileHiddenNameRules(names, pattern)
	return a.hidden
}

func compileHiddenNameRules(names, pattern string) *hiddenNameRules {
	rules := &hiddenNameRules{names: names, pattern: pattern, exact: make(map[string]bool)}
	for name := range strings.SplitSeq(names, ",") {
		if name = strings.TrimSpace(name); name != "" {
			rules.exact[name] = true
		}
	}
	for index, line := range strings.Split(strings.ReplaceAll(pattern, "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		compiled, err := regexp.Compile(line)
		if err != nil {
			rules.invalidLine = index + 1
			rules.regexps = nil
			break
		}
		rules.regexps = append(rules.regexps, compiled)
	}
	return rules
}

func (a *App) hiddenPath(path string) bool {
	rules := a.hiddenRules()
	if rules == nil || (len(rules.exact) == 0 && len(rules.regexps) == 0) {
		return false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(a.root, path)
	}
	rel, err := filepath.Rel(a.root, path)
	if err != nil || rel == "." {
		return false
	}
	for name := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
		if rules.exact[name] {
			return true
		}
		for _, pattern := range rules.regexps {
			if pattern.MatchString(name) {
				return true
			}
		}
	}
	return false
}
