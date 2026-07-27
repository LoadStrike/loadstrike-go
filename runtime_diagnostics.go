package loadstrike

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxRuntimeDiagnosticBytes = 4096

func sanitizeRuntimeDiagnostic(value string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			value = strings.ReplaceAll(value, secret, "[REDACTED]")
		}
	}

	value = strings.Map(func(character rune) rune {
		if character <= 0x1f ||
			(character >= 0x7f && character <= 0x9f) ||
			unicode.Is(unicode.Cf, character) ||
			unicode.IsSpace(character) {
			return ' '
		}
		return character
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= maxRuntimeDiagnosticBytes {
		return value
	}

	cut := maxRuntimeDiagnosticBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return strings.TrimSpace(value[:cut])
}
