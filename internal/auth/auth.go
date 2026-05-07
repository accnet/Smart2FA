package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	cookieName = "s2fa_session"
	cookieTTL  = 86400 * 7 // 7 days
)

// NewToken generates a random 32-byte hex session token.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SetCookie sets the session cookie on the response.
func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieTTL,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(cookieTTL * time.Second),
	})
}

// GetToken returns the session token from the request cookie.
func GetToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	if len(cookie.Value) != 64 {
		return "", false
	}
	return cookie.Value, true
}

// ClearCookie removes the session cookie.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    cookieName,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}
