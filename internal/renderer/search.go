package renderer

import (
	"bytes"
	"strconv"
	"strings"
	"unicode"

	"ports/internal/model"
)

// FilterRecords filters a slice of PortRecord based on a search query.
// It supports:
// - Exact, prefix, and substring matches on port numbers (e.g. "80", ":8080")
// - Case-insensitive substring and fuzzy subsequence matches on process names (e.g. "brave", "brv")
// - Protocol ("tcp", "udp") and user name matching
// - Multi-token queries (e.g. "node 3000", "tcp 53") where all tokens must match
func FilterRecords(records []model.PortRecord, query string) []model.PortRecord {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return records
	}

	tokens := strings.Fields(strings.ToLower(trimmed))
	if len(tokens) == 0 {
		return records
	}

	var matches []model.PortRecord
	for _, r := range records {
		if recordMatches(r, tokens) {
			matches = append(matches, r)
		}
	}
	return matches
}

func recordMatches(r model.PortRecord, tokens []string) bool {
	portStr := strconv.Itoa(int(r.Port))
	procLower := strings.ToLower(r.Process)
	protoLower := strings.ToLower(r.Protocol)
	userLower := ""
	if r.User != nil {
		userLower = strings.ToLower(*r.User)
	}

	for _, token := range tokens {
		cleanToken := strings.TrimPrefix(token, ":")
		isDigits := isAllDigits(cleanToken)
		matched := false

		if isDigits {
			// Numeric token: prioritize port number match
			if strings.Contains(portStr, cleanToken) {
				matched = true
			} else if strings.Contains(procLower, token) {
				matched = true
			}
		} else {
			// Text token: match process name (substring or fuzzy), protocol, or user
			if strings.Contains(procLower, token) {
				matched = true
			} else if isSubsequence(token, procLower) {
				matched = true
			} else if strings.EqualFold(protoLower, token) || strings.HasPrefix(protoLower, token) {
				matched = true
			} else if userLower != "" && (strings.Contains(userLower, token) || isSubsequence(token, userLower)) {
				matched = true
			}
		}

		if !matched {
			return false
		}
	}
	return true
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// isSubsequence checks whether pattern runes appear in target in sequential order.
func isSubsequence(pattern, target string) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(pattern) > len(target) {
		return false
	}
	pIdx := 0
	pRunes := []rune(pattern)
	for _, r := range target {
		if r == pRunes[pIdx] {
			pIdx++
			if pIdx == len(pRunes) {
				return true
			}
		}
	}
	return false
}

// HighlightMatches highlights characters in s that match search tokens.
func HighlightMatches(s string, query string, theme *Theme, baseColor string) string {
	if !theme.Enabled || strings.TrimSpace(query) == "" {
		if theme.Enabled {
			return baseColor + s + theme.Reset
		}
		return s
	}

	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(tokens) == 0 {
		return baseColor + s + theme.Reset
	}

	sRunes := []rune(s)
	sLower := strings.ToLower(s)
	sLowerRunes := []rune(sLower)
	matched := make([]bool, len(sRunes))

	for _, token := range tokens {
		clean := strings.TrimPrefix(token, ":")
		if clean == "" {
			continue
		}
		cRunes := []rune(clean)
		idx := strings.Index(sLower, clean)
		if idx != -1 {
			rIdx := len([]rune(sLower[:idx]))
			for i := 0; i < len(cRunes) && rIdx+i < len(matched); i++ {
				matched[rIdx+i] = true
			}
		} else if isSubsequence(clean, sLower) {
			pIdx := 0
			for i, r := range sLowerRunes {
				if pIdx < len(cRunes) && r == cRunes[pIdx] {
					matched[i] = true
					pIdx++
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString(baseColor)
	inHighlight := false

	for i, r := range sRunes {
		if matched[i] && !inHighlight {
			buf.WriteString(theme.Reset + theme.Bold + theme.BrightYellow)
			inHighlight = true
		} else if !matched[i] && inHighlight {
			buf.WriteString(theme.Reset + baseColor)
			inHighlight = false
		}
		buf.WriteRune(r)
	}
	buf.WriteString(theme.Reset)
	return buf.String()
}
