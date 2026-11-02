package telegram

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"clientpulse/services/api/internal/notification"
)

const (
	// TelegramMaxMessageChars is the hard limit defined by Telegram Bot API sendMessage.
	TelegramMaxMessageChars = 4096

	// TelegramSafeMarginChars is the safety threshold to ensure messages remain comfortably
	// under the hard limit even after entity parsing and formatting tags.
	TelegramSafeMarginChars = 3800
)

// TelegramMessageLength returns the number of UTF-16 code units in s,
// which is the official metric Telegram Bot API uses for entity offsets and length limits.
func TelegramMessageLength(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// SafeTruncateRunes truncates string s to at most maxRunes, appending suffix if truncated.
// It ensures input is valid UTF-8 and never produces half-surrogates or broken multibyte sequences.
func SafeTruncateRunes(s string, maxRunes int, suffix string) string {
	valid := strings.ToValidUTF8(s, "")
	runes := []rune(valid)
	if len(runes) <= maxRunes {
		return valid
	}
	suffixRunes := []rune(suffix)
	take := maxRunes - len(suffixRunes)
	if take < 0 {
		take = 0
	}
	return string(runes[:take]) + suffix
}

// TruncateTargetURL safely truncates target URLs to a maximum rune length,
// ensuring secrets and query parameters are stripped before truncation.
func TruncateTargetURL(rawURL string, maxRunes int) string {
	sanitized := notification.SanitizeTargetURL(rawURL)
	if sanitized == "" {
		return ""
	}

	runes := []rune(sanitized)
	if len(runes) <= maxRunes {
		return sanitized
	}

	// If URL is too long, strip query entirely before truncating to guarantee no truncated tokens leak
	parsed, err := url.Parse(sanitized)
	if err == nil {
		parsed.RawQuery = ""
		clean := parsed.String()
		if len([]rune(clean)) <= maxRunes {
			return clean
		}
		sanitized = clean
	}

	return SafeTruncateRunes(sanitized, maxRunes, "...")
}

var tagRegex = regexp.MustCompile(`^</?([a-zA-Z0-9_-]+)(?:\s+[^>]*)?>`)

// EnsureValidTelegramHTML validates, truncates if necessary, and guarantees all opened HTML
// tags are properly closed without creating malformed entities or unbalanced tags.
func EnsureValidTelegramHTML(htmlText string, maxChars int) string {
	if maxChars <= 0 {
		maxChars = TelegramMaxMessageChars
	}

	// 1. Ensure valid UTF-8
	validUTF8 := strings.ToValidUTF8(htmlText, "")

	var result strings.Builder
	var tagStack []string

	runes := []rune(validUTF8)
	n := len(runes)
	i := 0

	for i < n {
		// Calculate length needed to close all currently opened tags plus optional ellipsis
		closingLen := 0
		for _, tag := range tagStack {
			closingLen += len(tag) + 3 // "</" + tag + ">"
		}

		// Check if adding more content would exceed maxChars
		currentLen := TelegramMessageLength(result.String())
		if currentLen+closingLen+3 >= maxChars { // 3 for "..."
			result.WriteString("...")
			break
		}

		if runes[i] == '<' {
			// Find tag end
			sub := string(runes[i:])
			loc := tagRegex.FindStringSubmatchIndex(sub)
			if loc != nil {
				tagFull := sub[loc[0]:loc[1]]
				tagName := strings.ToLower(sub[loc[2]:loc[3]])
				isClosing := strings.HasPrefix(tagFull, "</")

				// If appending this full tag would exceed maxChars, stop
				if currentLen+TelegramMessageLength(tagFull)+closingLen >= maxChars {
					result.WriteString("...")
					break
				}

				if isClosing {
					// Pop matching tag or unwind stack
					if len(tagStack) > 0 && tagStack[len(tagStack)-1] == tagName {
						tagStack = tagStack[:len(tagStack)-1]
						result.WriteString(tagFull)
					} else {
						// Look deeper in stack
						foundIdx := -1
						for sIdx := len(tagStack) - 1; sIdx >= 0; sIdx-- {
							if tagStack[sIdx] == tagName {
								foundIdx = sIdx
								break
							}
						}
						if foundIdx != -1 {
							// Close intermediate tags
							for sIdx := len(tagStack) - 1; sIdx >= foundIdx; sIdx-- {
								result.WriteString(fmt.Sprintf("</%s>", tagStack[sIdx]))
							}
							tagStack = tagStack[:foundIdx]
						}
					}
				} else {
					// Opening tag
					tagStack = append(tagStack, tagName)
					result.WriteString(tagFull)
				}

				// Advance index by tag rune length
				tagRuneLen := utf8.RuneCountInString(tagFull)
				i += tagRuneLen
				continue
			}
		}

		// Normal character
		result.WriteRune(runes[i])
		i++
	}

	// Close any unclosed tags in reverse order
	for idx := len(tagStack) - 1; idx >= 0; idx-- {
		result.WriteString(fmt.Sprintf("</%s>", tagStack[idx]))
	}

	return result.String()
}

// StripTelegramHTML strips Telegram HTML tags and returns formatted plain text.
func StripTelegramHTML(htmlText string) string {
	// Convert <a href="url">label</a> to "label (url)" or "label"
	aTagRegex := regexp.MustCompile(`(?i)<a\s+href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	plain := aTagRegex.ReplaceAllStringFunc(htmlText, func(match string) string {
		parts := aTagRegex.FindStringSubmatch(match)
		if len(parts) >= 3 {
			urlStr := parts[1]
			labelStr := parts[2]
			if labelStr == "" {
				return urlStr
			}
			return fmt.Sprintf("%s (%s)", labelStr, urlStr)
		}
		return match
	})

	// Strip remaining HTML tags
	stripRegex := regexp.MustCompile(`<[^>]+>`)
	plain = stripRegex.ReplaceAllString(plain, "")

	// Unescape HTML entities
	return html.UnescapeString(plain)
}
