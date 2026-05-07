package totp

import (
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

// GenerateCode returns the current TOTP code for a given secret.
func GenerateCode(secret string) (string, error) {
	// Strip ALL whitespace — users often paste secrets with spaces (e.g. "N6ZL GKOD 45XX...")
	secret = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, secret)
	code, err := totp.GenerateCode(strings.ToUpper(secret), time.Now())
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return code, nil
}

// TimeRemaining returns seconds remaining in the current 30s window.
func TimeRemaining() int {
	return 30 - int(time.Now().Unix()%30)
}

// ParseOTPAuth extracts name and secret from an otpauth:// URI.
// Returns name, secret, error.
func ParseOTPAuth(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		return "", "", fmt.Errorf("invalid otpauth URI")
	}
	// otpauth://totp/LABEL?secret=SECRET&...
	rest := strings.TrimPrefix(uri, "otpauth://totp/")
	parts := strings.SplitN(rest, "?", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid otpauth URI format")
	}
	label := parts[0]
	// label may be "issuer:account"
	if colonIdx := strings.LastIndex(label, ":"); colonIdx != -1 {
		label = label[colonIdx+1:]
	}
	label = strings.TrimSpace(label)

	// parse query params manually
	secret := ""
	for _, kv := range strings.Split(parts[1], "&") {
		kv = strings.TrimSpace(kv)
		if strings.HasPrefix(strings.ToLower(kv), "secret=") {
			secret = kv[7:]
		}
	}
	if secret == "" {
		return "", "", fmt.Errorf("secret not found in otpauth URI")
	}
	return label, secret, nil
}
