package server

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tdx/rkv/api"
	dbApi "github.com/tdx/rkv/db/api"
	clusterApi "github.com/tdx/rkv/internal/cluster/api"
	rpcApi "github.com/tdx/rkv/internal/rpc/v1"

	log "github.com/hashicorp/go-hclog"
)

// Server ...
type Server struct {
	joiner Joiner
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
	}
	config.Logger = logger

	s := &Server{
		db:     config.Db,
		logger: config.Logger,
		joiner: config.Joiner,
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
	case strings.HasPrefix(r.URL.Path, "/cluster/join"):
		s.clusterJoin(w, r)
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
	var (
		err  error
		keyb []byte
	)
	isHex, okHex := args["hex"]
	if okHex && isHex == "true" {
		keyb, err = hex.DecodeString(key)
		if err != nil {
			s.logger.Error("put bad hex", "key", key)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		keyb = []byte(key)
	}

	val, ok := args["val"]
	if !ok || val == "" {
		s.logger.Error("put argument error", "val", val)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var valb []byte
	if okHex && isHex == "true" {
		valb, err = hex.DecodeString(val)
		if err != nil {
			s.logger.Error("put bad hex", "val", val)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		valb = []byte(val)
	}

	if err := s.db.Put([]byte(tab), keyb, valb); err != nil {
		s.logger.Error("put", "error", err)
		switch err {
		case api.ErrNodeIsNotALeader:
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
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
	var (
		err  error
		keyb []byte
	)
	isHex, ok := args["hex"]
	if ok && isHex == "true" {
		keyb, err = hex.DecodeString(key)
		if err != nil {
			s.logger.Error("get bad hex", "key", key)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		keyb = []byte(key)
	}

	valb, err := s.db.Get(api.ReadLeader, []byte(tab), keyb)
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
		if err == api.ErrNodeIsNotALeader {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		s.logger.Error("get", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var val string
	if isHex == "true" {
		val = hex.EncodeToString(valb)
	} else {
		val = string(valb)
	}

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
	var (
		err  error
		keyb []byte
	)
	isHex, ok := args["hex"]
	if ok && isHex == "true" {
		keyb, err = hex.DecodeString(key)
		if err != nil {
			s.logger.Error("put bad hex", "key", key)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		keyb = []byte(key)
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
		if err == api.ErrNodeIsNotALeader {
			w.WriteHeader(http.StatusServiceUnavailable)
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
		s.logger.Error("membership RPCServers", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	grpcServers := make([]*rpcApi.Server, 0, len(servers))

	for i := range servers {
		server := servers[i]
		grpcServers = append(grpcServers, &rpcApi.Server{
			Id:       server.ID,
			Ip:       server.IP,
			Host:     server.Host,
			RpcPort:  server.RPCPort,
			RaftPort: server.RaftPort,
			IsLeader: server.IsLeader,
			Online:   server.Online,
		})
	}

	s.logger.Debug("Servers", "servers", grpcServers)

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
	host, ip := cluster.Leader()

	ret := `{"leader":{"host":"` + host + `","ip":"` + ip + `"}}`

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ret))
}

func (s *Server) clusterJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	args, ok := s.parseArgs(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	addrsStr, ok := args["addrs"]
	if !ok || addrsStr == "" {
		s.logger.Error("join argument error", "addrs", addrsStr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	addrs := strings.Split(addrsStr, ",")

	n, err := s.joiner.Join(addrs)
	if err != nil {
		s.logger.Error("join", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ret := `{"joined":"` + strconv.Itoa(n) + `"}`

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
