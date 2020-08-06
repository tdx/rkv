package server

import (
	remoteApi "rkv/internal/remote/api"

	log "github.com/hashicorp/go-hclog"
)

// Server ...
type Server struct {
	db     remoteApi.Backend
	logger log.Logger
	addr   string
}

// NewHTTPServer ...
func NewHTTPServer(config *Config) *Server {
	return &Server{
		db:     config.Db,
		logger: config.Logger,
		addr:   config.Addr,
	}
}

// TODO:

// // ServeHTTP allows to serve HTTP requests.
// func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	switch {
// 	case strings.HasPrefix(r.URL.Path, "/db/put"):
// 		s.put(w, r)
// 	case strings.HasPrefix(r.URL.Path, "/db/get"):
// 		s.get(w, r)
// 	case strings.HasPrefix(r.URL.Path, "/db/del"):
// 		s.del(w, r)
// 	default:
// 		w.WriteHeader(http.StatusNotFound)
// 	}
// }

// //
// //
// //
// type PutRequest struct {

// }

// func (s *Server) put(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "POST" {
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		return
// 	}

// }
