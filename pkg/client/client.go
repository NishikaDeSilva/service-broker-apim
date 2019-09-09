/*
 * Copyright (c) 2019 WSO2 Inc. (http:www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http:www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

// client package contains all function required to make API calls
package client

import (
	"bytes"
	"code.cloudfoundry.org/lager"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"time"
)

const (
	HeaderAuth                 = "Authorization"
	HeaderBear                 = "Bearer "
	ErrMSGUnableToCreateReq    = "unable to create request"
	ErrMSGUnableToParseReqBody = "unable to parse request body"
)

// Request wraps the http.request and the Body
// Body is wrapped with io.ReadSeeker which allows to reset the body buffer reader to initial state in retires
type Request struct {
	body io.ReadSeeker
	R    *http.Request
}

// RetryPolicy defines a function which validate the response and apply desired policy
// to determine whether to retry the particular request or not
type RetryPolicy func(resp *http.Response) bool

// BackOffPolicy policy determines the duration between two retires
type BackOffPolicy func(min, max time.Duration, attempt int) time.Duration

// Client represent the state of the HTTP client
type Client struct {
	httpClient    *http.Client
	checkForReTry RetryPolicy
	backOff       BackOffPolicy
	minBackOff    time.Duration
	maxBackOff    time.Duration
	// Maximum number of retries
	maxRetry int
}

var client = &Client{
	httpClient:    http.DefaultClient,
	checkForReTry: defaultRetryPolicy,
	backOff:       defaultBackOffPolicy,
	minBackOff:    1 * time.Second,
	maxBackOff:    10 * time.Second,
	maxRetry:      3,
}

// SetupClient overrides the default HTTP client. This method should be called before calling Invoke function
func SetupClient(c *http.Client) {
	client.httpClient = c
}

// Wraps more information about the error
type InvokeError struct {
	err        error
	StatusCode int
}

func (e *InvokeError) Error() string {
	return e.err.Error()
}

// B64BasicAuth returns a base64 encoded value of "u:p" string
// base64Encode("username:password")
func B64BasicAuth(u, p string) (string, error) {
	if u == "" || p == "" {
		return "", errors.Errorf(constants.ErrMSGInvalidParams, u, p)
	}
	d := u + ":" + p
	return base64.StdEncoding.EncodeToString([]byte(d)), nil
}

// ParseBody parse response body into the given struct
// Must send the pointer to the response body
// Returns any error occurred
func ParseBody(res *http.Response, v interface{}) error {
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

// Invoke the request and parse the response body to the given struct
// context parameter is used to maintain the request context in the log
// resCode parameter is used to determine the desired response code
// Returns any error occurred
func Invoke(context string, req *Request, body interface{}, resCode int) error {
	var resp *http.Response
	var err error
	resp, err = client.Do(req)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableInitiateReq, context)
	}
	if resp.StatusCode != resCode {
		return &InvokeError{
			err:        errors.Errorf(constants.ErrMSGUnsuccessfulAPICall, context, resp.Status, req.R.URL),
			StatusCode: resp.StatusCode,
		}
	}

	// If response has a body
	if body != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				utils.LogError(constants.ErrMSGUnableToCloseBody, err, &utils.LogData{})
			}
		}()

		err = ParseBody(resp, body)
		if err != nil {
			return &InvokeError{
				err:        errors.Wrapf(err, constants.ErrMSGUnableToParseRespBody, context),
				StatusCode: resp.StatusCode,
			}
		}
	}
	return nil
}

// PostReq creates a POST HTTP request with an Authorization header and set the content type to application/json
func PostReq(token, url string, body io.ReadSeeker) (*Request, error) {
	req, err := ToRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.R.Header.Add(HeaderAuth, HeaderBear+token)
	req.R.Header.Set(constants.HTTPContentType, constants.ContentTypeApplicationJson)
	return req, nil
}

// GetReq function creates a GET HTTP request with an Authorization header
func GetReq(token, url string) (*Request, error) {
	req, err := ToRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.R.Header.Add(HeaderAuth, HeaderBear+token)
	return req, nil
}

// DeleteReq function creates a DELETE HTTP request with an Authorization header
func DeleteReq(token, url string) (*Request, error) {
	req, err := ToRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.R.Header.Add(HeaderAuth, HeaderBear+token)
	return req, nil
}

// BodyReader returns the byte buffer representation of the provided struct
func BodyReader(v interface{}) (io.ReadSeeker, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToParseReqBody)
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// ToRequest function returns client.Request struct which wraps the http.request and the request Body
func ToRequest(method, url string, body io.ReadSeeker) (*Request, error) {
	var rcBody io.ReadCloser
	if body != nil {
		rcBody = ioutil.NopCloser(body)
	}
	req, err := http.NewRequest(method, url, rcBody)
	if err != nil {
		return nil, err
	}
	return &Request{R: req, body: body}, nil
}

// Do method invokes the request and returns the response and, an error if exists
// If the request is failed it will retry according to the registered Retry policy and Back off policy
func (c *Client) Do(req *Request) (resp *http.Response, err error) {
	for i := 1; i <= c.maxRetry; i++ {
		resp, err = c.httpClient.Do(req.R)
		// This error occurs due to  network connectivity problem and not for Non 2xx responses
		if err != nil {
			return nil, err
		}
		if !c.checkForReTry(resp) {
			break
		}
		var logData = &utils.LogData{
			Data: lager.Data{
				"url":           req.R.URL,
				"response code": resp.StatusCode,
			},
		}
		if req.body != nil {
			// Reset the body reader
			if _, err := req.body.Seek(0, 0); err != nil {
				utils.LogError("unable to reset body reader", err, logData)
				return nil, err
			}
		}
		bt := c.backOff(c.minBackOff, c.maxBackOff, i)
		logData.AddData("back off time", bt.Seconds()).AddData("attempt", i)
		utils.LogDebug("retrying the request", logData)
		time.Sleep(bt)
	}
	return resp, nil
}

// defaultRetryPolicy will retry the request if the response code is 4XX or 5XX
func defaultRetryPolicy(resp *http.Response) bool {
	if resp.StatusCode >= 400 {
		return true
	}
	return false
}

// defaultBackOffPolicy waits until attempt^2 or (min,max)
func defaultBackOffPolicy(min, max time.Duration, attempt int) time.Duration {
	du := math.Pow(2, float64(attempt))
	sleep := time.Duration(du) * time.Second
	if sleep < min {
		return min
	}
	if sleep > max {
		return max
	}
	return sleep
}
