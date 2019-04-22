/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// utils package contains all function required to make API calls
package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"io"
	"io/ioutil"
	"net/http"
)

const (
	HeaderAuth                 = "Authorization"
	HeaderBear                 = "Bearer "
	ErrMSGUnableToCreateReq    = "unable to create request"
	ErrMSGUnableToParseReqBody = "unable to parse request body"
)

var client = http.DefaultClient

// SetupClient overrides the default HTTP client. This method should be called before calling Invoke function
func SetupClient(c *http.Client) {
	client = c
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
func Invoke(context string, req *http.Request, body interface{}, resCode int) error {
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableInitiateReq, context)
	}
	if resp.StatusCode != resCode {
		return &InvokeError{
			err:        errors.Errorf(constants.ErrMSGUnsuccessfulAPICall, context, resp.Status),
			StatusCode: resp.StatusCode,
		}
	}
	// If response has a body
	if body != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				utils.LogError(constants.ErrMSGUnableToCloseBody, err)
			}
		}()
		if resp.StatusCode == resCode {
			err = ParseBody(resp, body)
			if err != nil {
				return &InvokeError{
					err:        errors.Wrapf(err, constants.ErrMSGUnableToParseRespBody, context),
					StatusCode: resp.StatusCode,
				}
			}
		}
	}
	return nil
}

// PostReq creates a POST HTTP request with an Authorization header and set the content type to application/json
func PostReq(token, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.Header.Add(HeaderAuth, HeaderBear+token)
	req.Header.Set(constants.HTTPContentType, constants.ContentTypeApplicationJson)
	return req, nil
}

// DeleteReq function creates a DELETE HTTP request with an Authorization header
func DeleteReq(token, url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToCreateReq)
	}
	req.Header.Add(HeaderAuth, HeaderBear+token)
	return req, nil
}

// ByteBuf returns the byte buffer representation of the provided struct
func ByteBuf(v interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)
	if err != nil {
		return nil, errors.Wrap(err, ErrMSGUnableToParseReqBody)
	}
	return buf, nil
}
