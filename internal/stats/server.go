package stats

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Server is the HTTP server for the stats dashboard.
type Server struct {
	db     *DB
	addr   string
	mux    *http.ServeMux
}

// NewServer creates a new stats server.
func NewServer(db *DB, addr string) *Server {
	s := &Server{
		db:   db,
		addr: addr,
		mux:  http.NewServeMux(),
	}
	s.routes()
	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("[stats] dashboard listening on http://%s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/summary", s.handleSummary)
	s.mux.HandleFunc("/api/timeseries", s.handleTimeSeries)
	s.mux.HandleFunc("/api/by-provider", s.handleByProvider)
	s.mux.HandleFunc("/api/by-model", s.handleByModel)
	s.mux.HandleFunc("/api/recent", s.handleRecent)
}

func (s *Server) parseQuery(r *http.Request) Query {
	q := Query{GroupBy: "day"}

	fromStr := r.URL.Query().Get("from")
	if fromStr != "" {
		if d, err := time.Parse("2006-01-02", fromStr); err == nil {
			q.From = d
		}
	}

	toStr := r.URL.Query().Get("to")
	if toStr != "" {
		if d, err := time.Parse("2006-01-02", toStr); err == nil {
			q.To = d.Add(24 * time.Hour)
		}
	}

	if vendor := r.URL.Query().Get("vendor"); vendor != "" {
		q.Vendor = vendor
	}
	if protocol := r.URL.Query().Get("protocol"); protocol != "" {
		q.Protocol = protocol
	}
	if model := r.URL.Query().Get("model"); model != "" {
		q.Model = model
	}
	if groupBy := r.URL.Query().Get("groupBy"); groupBy != "" {
		q.GroupBy = groupBy
	}

	return q
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	q := s.parseQuery(r)
	summary, err := s.db.Summary(q)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, summary)
}

func (s *Server) handleTimeSeries(w http.ResponseWriter, r *http.Request) {
	q := s.parseQuery(r)
	data, err := s.db.TimeSeries(q)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleByProvider(w http.ResponseWriter, r *http.Request) {
	q := s.parseQuery(r)
	data, err := s.db.ByProvider(q)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleByModel(w http.ResponseWriter, r *http.Request) {
	q := s.parseQuery(r)
	data, err := s.db.ByModel(q)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, data)
}

func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = n
		}
	}
	data, err := s.db.Recent(page, pageSize)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, data)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPartialContent)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}


