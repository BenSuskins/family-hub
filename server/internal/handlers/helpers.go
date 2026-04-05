package handlers

import "net/http"

const DateFormat = "2006-01-02"

func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
