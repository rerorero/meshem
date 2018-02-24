package ctlapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/rerorero/meshem/src/model"
)

// PostServiceReq is the request type of POST service method.
type PostServiceReq struct {
	Protocol string `json:"protocol"`
}

// PutServiceResp is  theresponse type of PUT service method.
type PutServiceResp struct {
	Changed bool `json:"changed"`
}

// postService is handler to create a new service.
func (srv *Server) postSerivce(w http.ResponseWriter, r *http.Request, param httprouter.Params, body []byte) {
	var req PostServiceReq
	if err := json.Unmarshal(body, &req); err != nil {
		srv.respondError(http.StatusBadRequest, w, err)
		return
	}

	service, err := srv.inventory.RegisterService(param.ByName("name"), req.Protocol)
	if err != nil {
		srv.respondError(http.StatusInternalServerError, w, err)
		return
	}

	srv.respondJson(http.StatusCreated, w, &service)
}

// PostService calls POST service.
func (client *APIClient) PostService(name string, req PostServiceReq) (service model.Service, status int, err error) {
	var body []byte
	status, body, err = client.Post(client.serviceURIof(name), req)
	if err != nil {
		return service, status, err
	}
	err = json.Unmarshal(body, &service)
	return service, status, err
}

// getSerivce is handler to get a service.
func (srv *Server) getSerivce(w http.ResponseWriter, r *http.Request, param httprouter.Params, _ []byte) {
	service, ok, err := srv.inventory.GetService(param.ByName("name"))
	if err != nil {
		srv.respondError(http.StatusInternalServerError, w, err)
		return
	}
	if !ok {
		srv.respondError(http.StatusNotFound, w, fmt.Errorf("not found"))
		return
	}

	hosts, err := srv.inventory.GetHostsOfService(service.Name)
	if err != nil {
		srv.respondError(http.StatusInternalServerError, w, err)
		return
	}
	res := model.NewIdempotentService(&service, hosts)
	srv.respondJson(http.StatusOK, w, res)
}

// GetService calls GET service.
func (client *APIClient) GetService(name string) (resp model.IdempotentServiceParam, status int, err error) {
	var body []byte
	status, body, err = client.Get(client.serviceURIof(name))
	if err != nil {
		return resp, status, err
	}
	err = json.Unmarshal(body, &resp)
	return resp, status, err
}

// putService is handler to create/update a service idempotently.
func (srv *Server) putSerivce(w http.ResponseWriter, r *http.Request, param httprouter.Params, body []byte) {
	var req model.IdempotentServiceParam
	if err := json.Unmarshal(body, &req); err != nil {
		srv.respondError(http.StatusBadRequest, w, err)
		return
	}

	changed, err := srv.inventory.IdempotentService(param.ByName("name"), req)
	if err != nil {
		srv.respondError(http.StatusInternalServerError, w, err)
		return
	}

	res := PutServiceResp{Changed: changed}
	srv.respondJson(http.StatusOK, w, &res)
}

// PutService calls PUT service.
func (client *APIClient) PutService(name string, req model.IdempotentServiceParam) (resp PutServiceResp, status int, err error) {
	var body []byte
	status, body, err = client.Put(client.serviceURIof(name), req)
	if err != nil {
		return resp, status, err
	}
	err = json.Unmarshal(body, &resp)
	return resp, status, err
}

func (client *APIClient) serviceURIof(name string) string {
	return fmt.Sprintf("%s/%s/%s", client.endpoint.String(), ServiceURI, name)
}
