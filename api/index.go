package handler

import (
	"net/http"
	"strings"

	"max_bot_api/pkg/bot"
	"max_bot_api/pkg/httpapi"
)

// Index serves the mini-app REST API and bot webhook through one Vercel function.
func Index(w http.ResponseWriter, r *http.Request) {
	store, err := getStore()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if path := r.URL.Query().Get("path"); path != "" {
		request := r.Clone(r.Context())
		request.URL.Path = "/api/" + strings.TrimLeft(path, "/")
		query := request.URL.Query()
		query.Del("path")
		request.URL.RawQuery = query.Encode()
		r = request
	}

	httpapi.New(store, bot.NewNotifier()).ServeHTTP(w, r)
}
