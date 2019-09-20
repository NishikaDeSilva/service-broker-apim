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

// Package client contains functions required to make HTTP calls.
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/log"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"time"
)

const (
	HeaderAuth                  = "Authorization"
	HeaderBear                  = "Bearer "
	ErrMSGUnableToCreateReq     = "unable to create request"
	ErrMSGUnableToParseReqBody  = "unable to parse request body"
	ErrMSGUnableToParseRespBody = "unable to parse response body: %s "
	ErrMSGUnableInitiateReq     = "unable to initiate request: %s"
	ErrMSGUnsuccessfulAPICall   = "unsuccessful API call: %s response Code: %s URL: %s"
	ErrMSGUnableToCloseBody     = "unable to close the body"
	HTTPContentType             = "Content-Type"
	ContentTypeApplicationJSON  = "application/json"
	ContentTypeURLEncoded       = "application/x-www-form-urlencoded; param=value"
)

var ErrInvalidParameters = errors.New("invalid parameters")

// RetryPolicy defines a function which validate the response and apply desired policy
// to determine whether to retry the particular request or not.
type RetryPolicy func(resp *http.Response) bool

// BackOffPolicy policy determines the duration between two retires
type BackOffPolicy func(min, max time.Duration, attempt int) time.Duration

// Client represent the state of the HTTP client.
type Client struct {
	httpClient    *http.Client
	checkForReTry RetryPolicy
	backOff       BackOffPolicy
	minBackOff    time.Duration
	maxBackOff    time.Duration
	maxRetry      int
}

// default client
var client = &Client{
	httpClient:    http.DefaultClient,
	checkForReTry: defaultRetryPolicy,
	backOff:       defaultBackOffPolicy,
	minBackOff:    1 * time.Second,
	maxBackOff:    60 * time.Second,
	maxRetry:      3,
}

// HTTPRequestWrapper wraps the http.request and the Body.
// Body is wrapped with io.ReadSeeker which allows to reset the body buffer reader to initial state in retires.
type HTTPRequestWrapper struct {
	body    io.ReadSeeker
	httpReq *http.Request
}

// HTTPRequest returns the HTTP request.
func (r *HTTPRequestWrapper) HTTPRequest() *http.Request {
	return r.httpReq
}

// SetHeader method set the given header key and value to the HTTP request.
func (r *HTTPRequestWrapper) SetHeader(k, v string) {
	r.httpReq.Header.Set(k, v)
}

// Configure overrides the default client values. This method should be called before calling Invoke method.
func Configure(c *config.Client) {
	client = &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(c.Timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureCon},
			},
		},
		minBackOff:    time.Duration(c.MinBackOff) * time.Second,
		maxBackOff:    time.Duration(c.MaxBackOff) * time.Second,
		// to make sure that maxRetry is always a positive int
		maxRetry:      validMaxRetries(c.MaxRetries),
		backOff:       defaultBackOffPolicy,
		checkForReTry: defaultRetryPolicy,
	}
}

// InvokeError wraps more information about the error.
type InvokeError struct {
	err        error
	StatusCode int
}

func (e *InvokeError) Error() string {
	return e.err.Error()
}

// B64BasicAuth returns a base64 encoded value of "u:p" string and any error encountered.
func B64BasicAuth(u, p string) (string, error) {
	if u == "" || p == "" {
		return "", ErrInvalidParameters
	}
	d := u + ":" + p
	return base64.StdEncoding.EncodeToString([]byte(d)), nil
}

// ParseBody parse response body into the given struct.
// Must send the pointer to the response body.
// Returns any error encountered.
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

// Invoke the request and parse the response body to the given struct.
// context parameter is used to maintain the request context in the log.
// resCode parameter is used to determine the desired response code.
// Returns any error encountered.
func Invoke(context string, req *HTTPRequestWrapper, body interface{}, resCode int) error {
	var resp *http.Response
	var err error
	resp, err = do(req)
	if err != nil {
		return errors.Wrapf(err, ErrMSGUnableInitiateReq, context)
	}
	if resp.StatusCode != resCode {
		return &InvokeError{
			err:        errors.Errorf(ErrMSGUnsuccessfulAPICall, context, resp.Status, req.httpReq.URL),
			StatusCode: resp.StatusCode,
		}
	}

	// If response has a body
	if body != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Error(ErrMSGUnableToCloseBody, err, &log.Data{})
			}
		}()

		err = ParseBody(resp, body)
		if err != nil {
			return &InvokeError{
				err:        errors.Wrapf(err, ErrMSGUnableToParseRespBody, context),
				StatusCode: resp.StatusCode,
			}
		}
	}
	return nil
}

// PostHTTPRequestWrapper returns a POST HTTP request with a Bearer token header with the content type to application/json
// and any error encountered.
func PostHTTPRequestWrapper(token, url string, body io.ReadSeeker) (*HTTPRequestWrapper, error) {
	req, err := ToHTTPRequestWrapper(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.SetHeader(HeaderAuth, HeaderBear+token)
	req.SetHeader(HTTPContentType, ContentTypeApplicationJSON)
	return req, nil
}

// PutHTTPRequestWrapper returns a PUT HTTP request with a Bearer token header with the content type to application/json
// and any error encountered.
func PutHTTPRequestWrapper(token, url string, body io.ReadSeeker) (*HTTPRequestWrapper, error) {
	req, err := ToHTTPRequestWrapper(http.MethodPut, url, body)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.SetHeader(HeaderAuth, HeaderBear+token)
	req.SetHeader(HTTPContentType, ContentTypeApplicationJSON)
	return req, nil
}

// GetHTTPRequestWrapper returns a GET HTTP request with a Bearer token header
// and any error encountered.
func GetHTTPRequestWrapper(token, url string) (*HTTPRequestWrapper, error) {
	req, err := ToHTTPRequestWrapper(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.SetHeader(HeaderAuth, HeaderBear+token)
	return req, nil
}

// DeleteHTTPRequestWrapper returns a DELETE HTTP request with a Bearer token header
// and any error encountered.
func DeleteHTTPRequestWrapper(token, url string) (*HTTPRequestWrapper, error) {
	req, err := ToHTTPRequestWrapper(http.MethodDelete, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.SetHeader(HeaderAuth, HeaderBear+token)
	return req, nil
}

// BodyReader returns the byte buffer representation of the provided struct and any error encountered.
func BodyReader(v interface{}) (io.ReadSeeker, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToParseReqBody)
	}
	return bytes.NewReader(buf.Bytes()), nil
}

// ToHTTPRequestWrapper returns client.HTTPRequestWrapper struct which wraps the http.request, request Body any error encountered.
func ToHTTPRequestWrapper(method, url string, body io.ReadSeeker) (*HTTPRequestWrapper, error) {
	var rcBody io.ReadCloser
	if body != nil {
		rcBody = ioutil.NopCloser(body)
	}
	req, err := http.NewRequest(method, url, rcBody)
	if err != nil {
		return nil, err
	}
	return &HTTPRequestWrapper{httpReq: req, body: body}, nil
}

// do invokes the request and returns the response and, an error if exists.
// If the request is failed it will retry according to the registered Retry policy and Back off policy.
func do(reqWrapper *HTTPRequestWrapper) (resp *http.Response, err error) {
	for i := 1; i <= client.maxRetry; i++ {
		resp, err = client.httpClient.Do(reqWrapper.httpReq)
		// This error occurs due to  network connectivity problem and not for Non 2xx responses
		if err != nil {
			return nil, err
		}
		if !client.checkForReTry(resp) {
			break
		}

		logData := log.NewData().
			Add("url", reqWrapper.httpReq.URL).
			Add("response code", resp.StatusCode)
		if reqWrapper.body != nil {
			// Reset the body reader
			log.Debug("resetting the request body", logData)
			if _, err := reqWrapper.body.Seek(0, 0); err != nil {
				return nil, err
			}
		}
		bt := client.backOff(client.minBackOff, client.maxBackOff, i)
		logData.Add("back off time", bt.Seconds()).Add("attempt", i)
		log.Debug("retrying the request", logData)
		time.Sleep(bt)
	}
	return resp, nil
}

// defaultRetryPolicy will retry the request if the response code is 4XX or 5XX.
func defaultRetryPolicy(resp *http.Response) bool {
	if resp.StatusCode >= 400 {
		return true
	}
	return false
}

// defaultBackOffPolicy waits until attempt^2 or (min,max).
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

// validMaxRetries returns a valid maximum number of retires.
func validMaxRetries(maxRetries int) int {
	if maxRetries > 0 {
		return maxRetries
	}
	return 3
}
