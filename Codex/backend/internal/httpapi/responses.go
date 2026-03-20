package httpapi

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Error errorContent `json:"error"`
}

type errorContent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{
		Error: errorContent{
			Code:    code,
			Message: msg,
		},
	})
}
