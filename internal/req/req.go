package req

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/swahili-chess/notifier-bot/config"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func PostOrPutRequest(method string, url string, payload interface{}, errorResponse interface{}) (int, error) {

	if payload == nil || url == "" || errorResponse == nil {
		return 0, errors.New("postorputrequest: payload or errorresponse not provided")
	}

	b, err := json.Marshal(&payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(b))

	if err != nil {
		return 0, fmt.Errorf("postorputrequest: could not create request: %s", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(config.Cfg.BasicAuth.USERNAME, config.Cfg.BasicAuth.PASSWORD)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	r, err := client.Do(req)

	if err != nil {
		return 0, fmt.Errorf("postorputrequest: error making http request: %s", err)

	}

	if r.StatusCode == http.StatusInternalServerError {
		err := json.NewDecoder(r.Body).Decode(&errorResponse)
		return http.StatusInternalServerError, err

	} else if r.StatusCode == http.StatusOK {
		return http.StatusOK, nil

	} else if r.StatusCode == http.StatusBadRequest {
		return http.StatusBadRequest, errors.New("postorputrequest: bad request error")

	} else if r.StatusCode == http.StatusNotFound {
		return http.StatusNotFound, errors.New("postorputrequest: url not found")
	}

	return 0, errors.New("postorputrequest: unknown error")

}

func GetRequest(url string, response interface{}, errorResponse interface{}) (int, error) {

	if response == nil || url == "" || errorResponse == nil {
		return 0, errors.New("getrequest: response or errorresponse not provided")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return 0, fmt.Errorf("getrequest: could not create request: %s", err)
	}

	req.SetBasicAuth(config.Cfg.BasicAuth.USERNAME, config.Cfg.BasicAuth.PASSWORD)

	client := http.Client{
		Timeout: 10 * time.Second,
	}

	r, err := client.Do(req)

	if err != nil {
		return 0, fmt.Errorf("getrequest: error making http request: %s", err)

	}

	if r.StatusCode == http.StatusInternalServerError {
		err := json.NewDecoder(r.Body).Decode(&errorResponse)
		return http.StatusInternalServerError, err

	} else if r.StatusCode == http.StatusOK {
		err := json.NewDecoder(r.Body).Decode(&response)
		return http.StatusOK, err

	} else if r.StatusCode == http.StatusBadRequest {
		return http.StatusBadRequest, errors.New("getrequest: bad request error")

	} else if r.StatusCode == http.StatusNotFound {
		return http.StatusNotFound, errors.New("getrequest: url not found")
	}

	return 0, errors.New("getrequest: unknown error")

}
