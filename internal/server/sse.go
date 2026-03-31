package server

import "net/http"

type SSEHub struct {
	root string
}

func NewSSEHub(root string) *SSEHub {
	return &SSEHub{root: root}
}

func (h *SSEHub) Start() error {
	return nil
}

func (h *SSEHub) Stop() {}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
