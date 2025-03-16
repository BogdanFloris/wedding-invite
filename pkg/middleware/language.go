package middleware

import (
	"context"
	"net/http"
	"time"
	"wedding-invite/pkg/i18n"
)

const (
	// LanguageKey is the context key for language
	LanguageKey contextKey = "language"
)

// Language middleware extracts language from cookie or URL and adds it to context
func Language(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var lang string

		// Check for language cookie
		langCookie, err := r.Cookie("lang")
		if err == nil {
			lang = i18n.GetLanguage(langCookie.Value)
		} else {
			lang = i18n.DefaultLanguage
		}

		// Check if there's a language change request
		if r.URL.Query().Get("lang") != "" {
			newLang := r.URL.Query().Get("lang")
			if newLang == "en" || newLang == "ro" {
				lang = newLang

				// Set language cookie
				cookie := &http.Cookie{
					Name:     "lang",
					Value:    lang,
					Path:     "/",
					HttpOnly: true,
					Secure:   r.TLS != nil,
					SameSite: http.SameSiteLaxMode,
					Expires:  time.Now().Add(365 * 24 * time.Hour), // 1 year
				}
				http.SetCookie(w, cookie)
			}
		}

		// Add language to context
		ctx := context.WithValue(r.Context(), LanguageKey, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetLanguage extracts language from request context
func GetLanguage(r *http.Request) string {
	lang, ok := r.Context().Value(LanguageKey).(string)
	if !ok {
		return i18n.DefaultLanguage
	}
	return lang
}
