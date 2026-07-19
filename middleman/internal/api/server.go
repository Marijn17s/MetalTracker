package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"MetalTracker/middleman/internal/service"
)

type Server struct {
	service   *service.Service
	apiKey    string
	mux       *http.ServeMux
}

func New(svc *service.Service, middlemanAPIKey string) *Server {
	server := &Server{
		service: svc,
		apiKey:  strings.TrimSpace(middlemanAPIKey),
		mux:     http.NewServeMux(),
	}
	server.routes()
	return server
}

func (server *Server) Handler() http.Handler {
	return server.withAuth(server.mux)
}

func (server *Server) routes() {
	server.mux.HandleFunc("/healthz", server.handleHealth)
	server.mux.HandleFunc("/v1/latest", server.handleLatest)
	server.mux.HandleFunc("/v1/timeframe", server.handleTimeframe)
	server.mux.HandleFunc("/v1/", server.handleDateOrNotFound)
}

func (server *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if server.apiKey == "" || request.URL.Path == "/healthz" {
			next.ServeHTTP(writer, request)
			return
		}
		provided := request.Header.Get("X-API-Key")
		if provided == "" {
			provided = request.URL.Query().Get("api_key")
		}
		if provided != server.apiKey {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (server *Server) handleHealth(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) handleLatest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	base, currencies := parseQuoteQuery(request)
	payload, err := server.service.Latest(request.Context(), base, currencies)
	if err != nil {
		writeError(writer, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

func (server *Server) handleTimeframe(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	base, currencies := parseQuoteQuery(request)
	startText := request.URL.Query().Get("start_date")
	endText := request.URL.Query().Get("end_date")
	from, err := time.Parse("2006-01-02", startText)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid start_date")
		return
	}
	to, err := time.Parse("2006-01-02", endText)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid end_date")
		return
	}
	payload, err := server.service.Timeframe(request.Context(), from, to, base, currencies)
	if err != nil {
		writeError(writer, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

func (server *Server) handleDateOrNotFound(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeError(writer, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(request.URL.Path, "/v1/")
	path = strings.Trim(path, "/")
	if path == "" || path == "latest" || path == "timeframe" {
		writeError(writer, http.StatusNotFound, "not found")
		return
	}
	day, err := time.Parse("2006-01-02", path)
	if err != nil {
		writeError(writer, http.StatusNotFound, "not found")
		return
	}
	base, currencies := parseQuoteQuery(request)
	payload, err := server.service.Historical(request.Context(), day, base, currencies)
	if err != nil {
		writeError(writer, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, payload)
}

func parseQuoteQuery(request *http.Request) (string, []string) {
	base := request.URL.Query().Get("base")
	if base == "" {
		base = "EUR"
	}
	raw := request.URL.Query().Get("currencies")
	parts := strings.Split(raw, ",")
	currencies := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			currencies = append(currencies, part)
		}
	}
	return base, currencies
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

// ListenAddrFromEnv returns LISTEN_ADDR or :8080.
func ListenAddrFromEnv() string {
	if value := strings.TrimSpace(os.Getenv("LISTEN_ADDR")); value != "" {
		return value
	}
	return ":8080"
}
