package server

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"

	log "github.com/hashicorp/go-hclog"
)

// Server ...
type Server struct {
	db     clusterApi.Backend
	logger log.Logger
}

type arguments map[string]string

// NewHTTPServer ...
func NewHTTPServer(config *Config) (*http.Server, error) {

	// check logger
	logger := config.Logger
	if logger == nil {
		logger = log.New(&log.LoggerOptions{
			Name:  "http",
			Level: log.Error,
		})
	} else {
		logger = logger.Named("http")
	}
	config.Logger = logger

	s := &Server{
		db:     config.Db,
		logger: config.Logger,
	}

	return &http.Server{
		Handler:      s,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
	}, nil
}

// ServeHTTP allows to serve HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/db/put"):
		s.dbPut(w, r)
	case strings.HasPrefix(r.URL.Path, "/db/get"):
		s.dbGet(w, r)
	case strings.HasPrefix(r.URL.Path, "/db/del"):
		s.dbDel(w, r)
	case strings.HasPrefix(r.URL.Path, "/cluster/servers"):
		s.clusterServers(w, r)
	case strings.HasPrefix(r.URL.Path, "/cluster/leader"):
		s.clusterLeader(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

//
// DB
//
func (s *Server) dbPut(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	args, ok := s.parseArgs(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tab, ok := args["tab"]
	if !ok || tab == "" {
		s.logger.Error("put argument error", "tab", tab)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key, ok := args["key"]
	if !ok || key == "" {
		s.logger.Error("put argument error", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	keyb, err := hex.DecodeString(key)
	if err != nil {
		s.logger.Error("put bad hex", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	val, ok := args["val"]
	if !ok || val == "" {
		s.logger.Error("put argument error", "val", val)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	valb, err := hex.DecodeString(val)
	if err != nil {
		s.logger.Error("put bad hex", "val", val)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.db.Put([]byte(tab), keyb, valb); err != nil {
		s.logger.Error("put", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := `{"result":"ok"}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

func (s *Server) dbGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	args, ok := s.parseArgs(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tab, ok := args["tab"]
	if !ok || tab == "" {
		s.logger.Error("get argument error", "tab", tab)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key, ok := args["key"]
	if !ok || key == "" {
		s.logger.Error("get argument error", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	keyb, err := hex.DecodeString(key)
	if err != nil {
		s.logger.Error("get bad hex", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	valb, err := s.db.Get(api.ReadAny, []byte(tab), keyb)
	if err != nil {
		if dbApi.IsNoTableError(err) {
			s.logger.Error("not exists", "tab", tab)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if dbApi.IsNoKeyError(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.logger.Error("get", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	val := hex.EncodeToString(valb)

	ret := `{"val":"` + val + `"}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

func (s *Server) dbDel(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" && r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	args, ok := s.parseArgs(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tab, ok := args["tab"]
	if !ok || tab == "" {
		s.logger.Error("put argument error", "tab", tab)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key, ok := args["key"]
	if !ok || key == "" {
		s.logger.Error("put argument error", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	keyb, err := hex.DecodeString(key)
	if err != nil {
		s.logger.Error("put bad hex", "key", key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.db.Delete([]byte(tab), keyb); err != nil {
		if dbApi.IsNoTableError(err) {
			s.logger.Error("not exists", "tab", tab)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if dbApi.IsNoKeyError(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s.logger.Error("put", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := `{"result":"ok"}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

//
// Cluster
//
func (s *Server) clusterServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cluster := s.db.(clusterApi.Cluster)
	servers, err := cluster.Servers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret, err := json.Marshal(servers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

func (s *Server) clusterLeader(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cluster := s.db.(clusterApi.Cluster)
	leader := cluster.LeaderAddr()

	ret := `{"leader":"` + leader + `"}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

//
//
//
func (s *Server) parseArgs(r *http.Request) (arguments, bool) {

	var args arguments

	if r.Method == http.MethodGet {

		args = make(arguments)
		for k, v := range r.URL.Query() {
			args[k] = v[0]
		}

	} else if r.Method == http.MethodPost {

		args = s.getPostArgs(r)

	} else {

		return args, false
	}

	return args, true
}

func (s *Server) getPostArgs(r *http.Request) arguments {
	tag := "HTTP parse POST"
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		if err := r.ParseForm(); err != nil {
			s.logger.Error(tag, "error", err)
			return nil
		}
		args := make(map[string]string, len(r.PostForm))
		for k, v := range r.PostForm {
			args[k] = v[0]
		}
		return args
	}

	if r.Header.Get("Content-Type") == "application/json" {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			s.logger.Error(tag, "read_body", err)
			return nil
		}
		// decoder := json.NewDecoder(r.Body)
		// err := decoder.Decode(&args)

		args := make(arguments)
		err = json.Unmarshal(body, &args)
		if err != nil {
			s.logger.Error(tag, "parse_json", err)
			return nil
		}

		return args
	}

	return nil
}
