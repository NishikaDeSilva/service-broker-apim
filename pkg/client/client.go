/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// utils package contains all function required to make API calls
package client

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"io/ioutil"
	"net/http"
)

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
func Invoke(insecureCon bool, context string, req *http.Request, body interface{}, resCode int) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureCon},
	}
	client := &http.Client{
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableInitiateReq, context)
	}
	if resp.StatusCode != resCode {
		return errors.Errorf(constants.ErrMSGUnsuccessfulAPICall, resp.Status)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			utils.LogError(constants.ErrMSGUnableToCloseBody, err)
		}
	}()
	if resp.StatusCode == resCode {
		err = ParseBody(resp, body)
		if err != nil {
			return errors.Wrapf(err, constants.ErrMSGUnableToParseRespBody, context)
		}
	}
	return nil
}
