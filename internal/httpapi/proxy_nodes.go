package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxynode"
)

const maxProxyNodeBodyBytes int64 = 8 << 10

type proxyNodePayload struct {
	Type                proxynode.Type `json:"node_type"`
	Name                string         `json:"name"`
	Region              string         `json:"region"`
	Host                string         `json:"host"`
	PublicHost          string         `json:"public_host"`
	MTProtoPort         *int           `json:"mtproto_port"`
	InternalAPIEndpoint string         `json:"internal_api_endpoint"`
	Enabled             *bool          `json:"enabled"`
}

func (s *Server) registerProxyNodeRoutes() {
	s.mux.HandleFunc("GET /api/nodes", s.handleProxyNodeList)
	s.mux.HandleFunc("POST /api/nodes", s.handleProxyNodeCreate)
	s.mux.HandleFunc("PUT /api/nodes/{id}", s.handleProxyNodeUpdate)
	s.mux.HandleFunc("DELETE /api/nodes/{id}", s.handleProxyNodeDelete)
}

func (s *Server) handleProxyNodeList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	nodes, err := proxynode.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

func (s *Server) handleProxyNodeCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	input, ok := decodeProxyNodeInput(w, r)
	if !ok {
		return
	}
	node, err := proxynode.Create(r.Context(), s.db, input, time.Now().UTC())
	switch {
	case errors.Is(err, proxynode.ErrConflict):
		writeProxyNodeProblem(w, r, http.StatusConflict, "PROXY_NODE_CONFLICT", "A Proxy Node with that name already exists.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"node": node})
}

func (s *Server) handleProxyNodeUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := proxyNodeID(w, r)
	if !ok {
		return
	}
	input, ok := decodeProxyNodeInput(w, r)
	if !ok {
		return
	}
	node, err := proxynode.Update(r.Context(), s.db, id, input, time.Now().UTC())
	switch {
	case errors.Is(err, proxynode.ErrNotFound):
		writeProxyNodeProblem(w, r, http.StatusNotFound, "PROXY_NODE_NOT_FOUND", "The Proxy Node was not found.")
		return
	case errors.Is(err, proxynode.ErrConflict):
		writeProxyNodeProblem(w, r, http.StatusConflict, "PROXY_NODE_CONFLICT", "A Proxy Node with that name already exists.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"node": node})
}

func (s *Server) handleProxyNodeDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := proxyNodeID(w, r)
	if !ok {
		return
	}
	if err := proxynode.Delete(r.Context(), s.db, id); errors.Is(err, proxynode.ErrNotFound) {
		writeProxyNodeProblem(w, r, http.StatusNotFound, "PROXY_NODE_NOT_FOUND", "The Proxy Node was not found.")
		return
	} else if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeProxyNodeInput(w http.ResponseWriter, r *http.Request) (proxynode.CreateNode, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxProxyNodeBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload proxyNodePayload
	if err := decoder.Decode(&payload); err != nil {
		if isProxyNodeBodyTooLarge(err) {
			writeProxyNodeProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return proxynode.CreateNode{}, false
		}
		writeProxyNodeProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return proxynode.CreateNode{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isProxyNodeBodyTooLarge(err) {
			writeProxyNodeProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return proxynode.CreateNode{}, false
		}
		writeProxyNodeProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return proxynode.CreateNode{}, false
	}
	if payload.MTProtoPort == nil || payload.Enabled == nil {
		writeProxyNodeProblem(w, r, http.StatusBadRequest, "PROXY_NODE_INVALID", "The Proxy Node request is invalid.")
		return proxynode.CreateNode{}, false
	}
	input := proxynode.CreateNode{
		Type: payload.Type, Name: payload.Name, Region: payload.Region, Host: payload.Host, PublicHost: payload.PublicHost,
		MTProtoPort: *payload.MTProtoPort, InternalAPIEndpoint: payload.InternalAPIEndpoint, Enabled: *payload.Enabled,
	}
	if err := proxynode.ValidateInput(input); err != nil {
		writeProxyNodeProblem(w, r, http.StatusBadRequest, "PROXY_NODE_INVALID", "The Proxy Node request is invalid.")
		return proxynode.CreateNode{}, false
	}
	return input, true
}

func proxyNodeID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeProxyNodeProblem(w, r, http.StatusBadRequest, "PROXY_NODE_INVALID_ID", "The Proxy Node identifier is invalid.")
		return 0, false
	}
	return id, true
}

func isProxyNodeBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writeProxyNodeProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
