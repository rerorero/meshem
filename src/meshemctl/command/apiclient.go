package command

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rerorero/meshem/src/core/ctlapi"
	"github.com/rerorero/meshem/src/model"
)

// NewAPIClient creates a new APIClient inscance.
func NewAPIClient() (*ctlapi.APIClient, error) {
	endpoint := os.Getenv("MESHEM_CTLAPI_ENDPOINT")
	if len(endpoint) == 0 {
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", model.DefaultCtrlAPIPort)
	}

	timeout := os.Getenv("MESHEM_CTLAPI_TIMEOUT")
	if len(timeout) == 0 {
		timeout = "60"
	}
	t, err := strconv.Atoi(timeout)
	if err != nil {
		ExitWithError(errors.New("MESHEM_CTLAPI_TIMEOUT must be a number"))
	}
	timeoutDuration := time.Duration(t) * time.Second

	return ctlapi.NewClient(endpoint, timeoutDuration)
}
