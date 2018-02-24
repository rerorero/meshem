package ctlapi

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

// APIClient is client for ctlapi.
type APIClient struct {
	endpoint *url.URL
	client   http.Client
}

// NewClient returns a new ApiClient.
func NewClient(endpoint string, timeout time.Duration) (*APIClient, error) {
	client := http.Client{
		Timeout: timeout,
	}
	url, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid API endpoint: %s", endpoint)
	}
	return &APIClient{
		endpoint: url,
		client:   client,
	}, nil
}

// Post requests a POST method.
func (client *APIClient) Post(url string, body interface{}) (int, []byte, error) {
	return client.request(url, http.MethodPost, body)
}

// Put requests a PUT method.
func (client *APIClient) Put(url string, body interface{}) (int, []byte, error) {
	return client.request(url, http.MethodPut, body)
}

// Get requests a PUT method.
func (client *APIClient) Get(url string) (int, []byte, error) {
	return client.request(url, http.MethodGet, nil)
}

// Delete requests a DELETE method.
func (client *APIClient) Delete(url string) (int, []byte, error) {
	return client.request(url, http.MethodDelete, nil)
}

func (client *APIClient) request(url string, method string, body interface{}) (int, []byte, error) {
	var byte []byte
	var err error
	if body != nil {
		byte, err = json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(byte))
	if err != nil {
		return 0, nil, err
	}
	res, err := client.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, nil, err
	}
	return res.StatusCode, resBody, err
}
