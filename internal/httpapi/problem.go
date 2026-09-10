package httpapi

import (
	"encoding/json"
	"net/http"
)

type Problem struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Instance string         `json:"instance,omitempty"`
	TraceID  string         `json:"trace_id,omitempty"`
	Errors   []ProblemError `json:"errors,omitempty"`
}

type ProblemError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Field    string `json:"field,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
	Location string `json:"location,omitempty"`
}

func writeProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}
