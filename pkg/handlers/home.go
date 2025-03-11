package handlers

import (
	"net/http"
	"wedding-invite/templates"
)

func Home() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		templates.Home().Render(r.Context(), w)
	})
}