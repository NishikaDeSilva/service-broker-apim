/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
// utils package contains all function required to make API calls
package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"io/ioutil"
	"net/http"
	"net/url"
)

// DynamicClientRegReqBody represents the message body for Dynamic client request body
type DynamicClientRegReqBody struct {
	CallbackUrl string `json:"callbackUrl"`
	ClientName  string `json:"clientName"`
	Owner       string `json:"owner"`
	GrantType   string `json:"grantType"`
	SaasApp     bool   `json:"saasApp"`
}

// DynamicClientRegResBody represents the message body for Dynamic client response body
type DynamicClientRegResBody struct {
	CallbackUrl       string `json:"callBackURL"`
	JsonString        string `json:"jsonString"`
	ClientName        string `json:"clientName"`
	ClientId          string `json:"clientId"`
	ClientSecret      string `json:"clientSecret"`
	IsSaasApplication bool   `json:"isSaasApplication"`
}

// TokenResp represents the message body of the token api response
type TokenResp struct {
	Scope        string `json:"scope"`
	TokenTypes   string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// BasicCreds represents the username and password
type BasicCreds struct {
	Username string
	Password string
}

// DefaultClientRegBody returns a dynamic client request body with values
func DefaultClientRegBody() *DynamicClientRegReqBody {
	return &DynamicClientRegReqBody{
		CallbackUrl: constants.CallBackUrl,
		ClientName:  constants.ClientName,
		GrantType:   constants.DynamicClientRegGrantType,
		Owner:       constants.Owner,
		SaasApp:     true,
	}
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

func DynamicClientReg(uN, pW, url string, enableTLS bool, reqBody *DynamicClientRegReqBody) (er error,
	clientId, clientSecret string) {
	// Encode the body
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(reqBody)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableToParseRequestBody, "Dynamic Client Reg"), "", ""
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: enableTLS},
	}
	client := &http.Client{
		Transport: tr,
	}
	// construct the request
	req, err := http.NewRequest(http.MethodPost, url, b)
	req.SetBasicAuth(uN, pW)
	req.Header.Set(constants.HTTPContentType, constants.ContentTypeApplicationJson)

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableInitiateReq, "Dynamic Client Reg"), "", ""
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, constants.ErrMSGUnsuccessfulAPICall, "Dynamic Client Reg"), "", ""
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			utils.LogError(constants.ErrMSGUnableToCloseBody, err)
		}
	}()
	var body DynamicClientRegResBody
	if resp.StatusCode == http.StatusOK {
		err = ParseBody(resp, &body)
		if err != nil {
			return errors.Wrapf(err, constants.ErrMSGUnableToParseRespBody, "Dynamic Client Reg"), "", ""
		}
	}
	return nil, body.ClientId, body.ClientSecret
}

// GenAccessToken returns an Access token and a Refresh token from given params,
// 	Client credentials( Client key & Client secret)
// 	User credentials (Username & Password)
// 	Scope
// 	Endpoint
func GenAccessToken(clientCreds *BasicCreds, user *BasicCreds, urlS, scope string) (err error, aT, rT string) {
	data := url.Values{}
	data.Set(constants.UserName, user.Username)

	data.Add(constants.Password, user.Password)
	data.Add(constants.GrantType, constants.GrantPassword)
	data.Add(constants.Scope, scope)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}
	req, err := http.NewRequest(http.MethodPost, urlS, bytes.NewBufferString(data.Encode()))
	req.SetBasicAuth(clientCreds.Username, clientCreds.Password)
	req.Header.Add(constants.HTTPContentType, constants.ContentTypeUrlEncoded)

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, constants.ErrMSGUnableInitiateReq, "Generating access Token"), "", ""
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Wrapf(err, constants.ErrMSGUnsuccessfulAPICall, resp.StatusCode), "", ""
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			utils.LogError(constants.ErrMSGUnableToCloseBody, err)
		}
	}()
	var body TokenResp
	if resp.StatusCode == http.StatusOK {
		if err = ParseBody(resp, &body); err != nil {
			return errors.Wrapf(err, constants.ErrMSGUnableToParseRespBody, "Generating access Token"), "", ""
		}
	}
	return nil, body.AccessToken, body.RefreshToken
}
