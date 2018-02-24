package ctlapi

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/rerorero/meshem/src/core"
	"github.com/rerorero/meshem/src/model"
	"github.com/sirupsen/logrus"
)

// Server is a API server.
type Server struct {
	inventory      core.InventoryService
	router         *httprouter.Router
	conf           model.CtlAPIConf
	logger         *logrus.Logger
	bodyMaxbyteLen int64
}

// APIHandler is common HTTP handler type.
type APIHandler func(w http.ResponseWriter, r *http.Request, param httprouter.Params, body []byte)

type errorRes struct {
	Error string `json:"error"`
}

const (
	// ServiceURI is uri prefix for service resources.
	ServiceURI = "services"
)

// NewServer creates a new API server.
func NewServer(inventory core.InventoryService, conf model.CtlAPIConf, logger *logrus.Logger) *Server {
	srv := &Server{
		inventory:      inventory,
		router:         httprouter.New(),
		conf:           conf,
		logger:         logger,
		bodyMaxbyteLen: 1024 * 1024,
	}
	srv.router.POST(fmt.Sprintf("/%s/:name/", ServiceURI), srv.handlerOf(srv.postSerivce))
	srv.router.GET(fmt.Sprintf("/%s/:name/", ServiceURI), srv.handlerOf(srv.getSerivce))
	srv.router.PUT(fmt.Sprintf("/%s/:name/", ServiceURI), srv.handlerOf(srv.putSerivce))
	return srv
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.router.ServeHTTP(w, r)
}

// Run run
func (srv *Server) Run() error {
	go func() {
		srv.logger.Infof("ctlapi server listening on %d", srv.conf.Port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", srv.conf.Port), srv)
		if err != nil {
			srv.logger.Error("failed to start api server")
			srv.logger.Error(err)
		}
		srv.logger.Info("ctlapi server shutdown")
	}()
	return nil
}

func (srv *Server) respondJson(code int, w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		srv.logger.Errorf("ctlapi encode failed: %+v", body)
	}
}

func (srv *Server) respondError(code int, w http.ResponseWriter, err error) {
	srv.logger.Errorf("ctlapi error occurs(%d): %v", code, err)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	res := errorRes{Error: err.Error()}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		srv.logger.Errorf("ctlapi encode failed")
	}
}

func (srv *Server) authenticate(r *http.Request) error {
	// TODO
	return nil
}

// handlerOf is common handler process.
func (srv *Server) handlerOf(h APIHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, param httprouter.Params) {
		err := srv.authenticate(r)
		if err != nil {
			srv.respondError(http.StatusForbidden, w, err)
			return
		}
		body, err := ioutil.ReadAll(io.LimitReader(r.Body, srv.bodyMaxbyteLen))
		if err != nil {
			srv.respondError(http.StatusInternalServerError, w, errors.Wrap(err, "failed to read body"))
		}
		h(w, r, param, body)
	}
}
