package embedding

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	// MaxTextLength is the maximum length of text before truncation
	MaxTextLength = 8000
)

// PreprocessTextForEmbedding cleans and normalizes text before embedding
// This helps prevent NaN values in embedding models
func PreprocessTextForEmbedding(text string) string {
	if text == "" {
		return text
	}

	// Step 1: Remove control characters (except newlines and tabs)
	text = removeControlCharacters(text)

	// Step 2: Normalize whitespace
	text = normalizeWhitespace(text)

	// Step 3: Trim to max length
	if len(text) > MaxTextLength {
		text = truncateText(text, MaxTextLength)
	}

	// Step 4: Final trim
	text = strings.TrimSpace(text)

	return text
}

// removeControlCharacters removes Unicode control characters except newlines and tabs
func removeControlCharacters(text string) string {
	return strings.Map(func(r rune) rune {
		// Keep newlines, tabs, and printable characters
		if r == '\n' || r == '\t' || r == '\r' {
			return r
		}
		// Remove other control characters
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}

// normalizeWhitespace replaces multiple consecutive whitespace with single space
func normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`[ \t]+`)
	text = spaceRegex.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline (paragraph separator)
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	return text
}

// truncateText truncates text to maxLen, trying to break at sentence boundaries
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Try to break at sentence boundaries (period, exclamation, question mark)
	truncated := text[:maxLen]

	// Look for sentence endings in the last 100 characters
	searchStart := maxLen - 100
	if searchStart < 0 {
		searchStart = 0
	}

	sentenceEndings := []string{"。", ".", "！", "!", "？", "?", "\n"}
	bestPos := -1

	for _, ending := range sentenceEndings {
		if pos := strings.LastIndex(truncated[searchStart:], ending); pos != -1 {
			actualPos := searchStart + pos + len(ending)
			if actualPos > bestPos {
				bestPos = actualPos
			}
		}
	}

	if bestPos > searchStart {
		return text[:bestPos]
	}

	// If no sentence boundary found, just truncate
	return truncated
}

// TruncateTextWithRatio truncates text to a given ratio of its original length
// Used for retry logic when embedding fails
func TruncateTextWithRatio(text string, ratio float64) string {
	if ratio >= 1.0 {
		return text
	}

	newLen := int(float64(len(text)) * ratio)
	if newLen < 100 {
		newLen = 100 // Keep at least 100 characters
	}

	return truncateText(text, newLen)
}
