package analyzer

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func checkStartWithLowercase(message string) (bool, string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return false, ""
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	if unicode.IsLetter(r) && unicode.IsUpper(r) {
		newFirst := string(unicode.ToLower(r))
		rest := trimmed[utf8.RuneLen(r):]
		return true, newFirst + rest
	}
	return false, ""
}

func checkEnglishOnly(message string, cfg Config) (bool, string) {
	trimmed := strings.TrimSpace(message)
	var b strings.Builder
	for _, ch := range trimmed {
		if unicode.IsLetter(ch) && !unicode.In(ch, unicode.Latin) {
			continue
		}
		b.WriteRune(ch)
	}
	if b.String() != trimmed {
		return true, b.String()
	}
	return false, ""
}

var sensitive = []string{
	"password", "passwd", "pass", "api_key", "apikey", "api key", "api-key",
	"token", "secret", "ssn", "credit", "card", "cardnumber", "private key", "private_key",
}

func checkSensitiveKeys(message string, cfg Config) (bool, string) {
	low := strings.ToLower(message)
	for _, kw := range sensitive {
		if strings.Contains(low, kw) {
			return true, kw
		}
	}
	for _, pattern := range cfg.CustomSensitivePatterns {
		matched, err := regexp.MatchString(pattern, low)
		if err == nil && matched {
			return true, pattern
		}
	}
	return false, ""
}

func checkDisallowedSymbols(message string, cfg Config) (bool, string) {
	allowed := buildAllowedPunctuation(cfg)
	for _, ch := range message {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			continue
		}
		if allowed[ch] {
			continue
		}
		if unicode.IsControl(ch) || unicode.IsSymbol(ch) || unicode.IsMark(ch) || unicode.IsPunct(ch) {
			return true, string(ch)
		}
	}
	return false, ""
}

func buildAllowedPunctuation(cfg Config) map[rune]bool {
	allowed := make(map[rune]bool)
	for _, r := range cfg.AllowedPunctuation {
		allowed[r] = true
	}
	allowed[' '] = true
	return allowed
}
