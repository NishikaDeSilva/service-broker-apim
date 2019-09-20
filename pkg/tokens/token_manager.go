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

// Package tokens manages the Access tokens.
package tokens

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/log"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	SuffixSecond                  = "s"
	ErrMSGNotEnoughArgs           = "At least one scope should be present"
	ErrMSGUnableToGetClientCreds  = "Unable to get Client credentials"
	ErrMSGUnableToGetAccessToken  = "Unable to get access token for scope: %s"
	ErrMSGUnableToParseExpireTime = "Unable parse expiresIn time"
	GenerateAccessToken           = "Generating access Token"
	DynamicClientRegMSG           = "Dynamic Client Reg"
	RefreshTokenContext           = "Refresh token"
	DynamicClientContext          = "/client-registration/v0.14/register"
	TokenContext                    = "/token"
	UserName                        = "username"
	Password                        = "password"
	GrantPassword                   = "password"
	GrantRefreshToken               = "refresh_token"
	GrantType                       = "grant_type"
	Scope                           = "scope"
	RefreshToken                    = "refresh_token"
	ErrMSGUnableToParseRequestBody  = "unable to parse request body: %s"
	ErrMSGUnableToCreateRequestBody = "unable to create request body: %s"
	ScopeAPICreate                  = "apim:api_create"
	ScopeSubscribe                  = "apim:subscribe"
	ScopeAPIPublish                 = "apim:api_publish"
	ScopeAPIView                    = "apim:api_view"

	// CallBackURL is a dummy value
	CallBackURL = "www.dummy.com"

	// ClientName for dynamic client registration
	ClientName = "rest_api_publisher"

	// DynamicClientRegGrantType for dynamic client registration
	DynamicClientRegGrantType = "password refresh_token"

	// Owner for dynamic client registration
	Owner = "admin"
)

// BasicCredentials represents the username and Password.
type BasicCredentials struct {
	Username string
	Password string
}

// DynamicClientRegReq represents the request for Dynamic client request body.
type DynamicClientRegReq struct {
	CallbackURL string `json:"callbackUrl"`
	ClientName  string `json:"clientName"`
	Owner       string `json:"owner"`
	GrantType   string `json:"grantType"`
	SaasApp     bool   `json:"saasApp"`
}

// DynamicClientRegResBody represents the message body for Dynamic client registration response body.
type DynamicClientRegResBody struct {
	CallbackURL       string `json:"callBackURL"`
	JSONString        string `json:"jsonString"`
	ClientName        string `json:"clientName"`
	ClientID          string `json:"clientId"`
	ClientSecret      string `json:"clientSecret"`
	IsSaasApplication bool   `json:"isSaasApplication"`
}

// Resp represents the message body of the token api response.
type Resp struct {
	Scope        string `json:"scope"`
	TokenTypes   string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// tokens represent the Access token & Refresh token for a particular scope.
type tokens struct {
	lock      sync.RWMutex //ensures atomic writes to following fields
	aT        string
	rT        string
	expiresIn time.Time
}

// PasswordRefreshTokenGrantManager is used to manage Access token using password and refresh_token grant type.
// Holds access tokens for the given scopes and regenerate tokens using refresh tokens if the access token is expired.
type PasswordRefreshTokenGrantManager struct {
	once                  sync.Once
	holder                map[string]*tokens
	clientID              string
	clientSec             string
	TokenEndpoint         string
	DynamicClientEndpoint string
	UserName              string
	Password              string
}

// Manager interface manages the tokens for a set of given scopes.
type Manager interface {
	// InitTokenManager initialize the Token Manager. HTTPRequest tokens for the given scopes.
	// Must run before using the Token Manager.
	InitTokenManager(scopes ...string)

	// Token method returns a valid Access token.
	// Returns the access token and any error occurred.
	Token(scope string) (string, error)
}

// InitTokenManager initialize the Token Manager. HTTPRequest tokens for the given scopes.
// Must run before using the Token Manager.
func (m *PasswordRefreshTokenGrantManager) InitTokenManager(scopes ...string) {
	m.once.Do(func() {
		if len(scopes) == 0 {
			log.HandleErrorWithLoggerAndExit(ErrMSGNotEnoughArgs, nil)
		}

		var errDynamic error
		m.clientID, m.clientSec, errDynamic = m.dynamicClientReg(defaultClientRegBody())
		if errDynamic != nil {
			log.HandleErrorWithLoggerAndExit(ErrMSGUnableToGetClientCreds, errDynamic)
		}

		m.holder = make(map[string]*tokens)
		for _, scope := range scopes {
			data := m.accessTokenReqBody(scope)
			aT, rT, expiresIn, err := m.genToken(data, GenerateAccessToken)
			if err != nil {
				log.HandleErrorWithLoggerAndExit(fmt.Sprintf(ErrMSGUnableToGetAccessToken, scope), err)
			}
			// Handling the expire time of the access token
			duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + "s")
			if err != nil {
				log.HandleErrorWithLoggerAndExit(ErrMSGUnableToParseExpireTime, err)
			}
			m.holder[scope] = &tokens{
				aT:        aT,
				rT:        rT,
				expiresIn: time.Now().Add(duration),
			}
		}
	})
}

// accessTokenReqBody method returns access token request body.
func (m *PasswordRefreshTokenGrantManager) accessTokenReqBody(scope string) url.Values {
	data := url.Values{}
	data.Set(UserName, m.UserName)
	data.Add(Password, m.Password)
	data.Add(GrantType, GrantPassword)
	data.Add(Scope, scope)
	return data
}

// refreshTokenReqBody method returns refresh token request body.
func refreshTokenReqBody(rT string) url.Values {
	data := url.Values{}
	data.Add(RefreshToken, rT)
	data.Add(GrantType, GrantRefreshToken)
	return data
}

// isExpired method returns true if the difference between the current time and given time is 10s.
func isExpired(expiresIn time.Time) bool {
	if time.Now().Sub(expiresIn) > (10 * time.Second) {
		return true
	}
	return false
}

// Token method returns a valid Access token.
// Returns the access token and any error occurred.
func (m *PasswordRefreshTokenGrantManager) Token(scope string) (string, error) {
	t := m.holder[scope]
	t.lock.RLock()
	if !isExpired(t.expiresIn) {
		aT := t.aT
		t.lock.RUnlock()
		return aT, nil
	}
	t.lock.RUnlock()
	t.lock.Lock()
	if !isExpired(t.expiresIn) {
		t.lock.Unlock()
		return t.aT, nil
	}
	aT, rT, expiresIn, err := m.refreshToken(t.rT)
	if err != nil {
		t.lock.Unlock()
		return "", err
	}
	//Parse time to type time.Duration
	duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + SuffixSecond)
	ld := log.NewData().
		Add("access token", aT).
		Add("refresh token", rT).
		Add("expires in", expiresIn)
	log.Debug("token details", ld)
	m.holder[scope] = &tokens{
		aT:        aT,
		rT:        rT,
		expiresIn: time.Now().Add(duration),
	}
	t.lock.Unlock()
	return aT, nil
}

// refreshToken method generates a new Access token and a Refresh token.
func (m *PasswordRefreshTokenGrantManager) refreshToken(rTNow string) (aT, newRT string, expiresIn int, err error) {
	data := refreshTokenReqBody(rTNow)
	aT, rT, expiresIn, err := m.genToken(data, RefreshTokenContext)
	if err != nil {
		return "", "", 0, err
	}
	return aT, rT, expiresIn, nil
}

// genToken method returns an Access token and a Refresh token from given params.
func (m *PasswordRefreshTokenGrantManager) genToken(reqBody url.Values, context string) (aT, rT string, expiresIn int, err error) {
	u, err := utils.ConstructURL(m.TokenEndpoint, TokenContext)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "cannot construct, token endpoint")
	}
	rWrapper, err := client.ToHTTPRequestWrapper(http.MethodPost, u, bytes.NewReader([]byte(reqBody.Encode())))
	if err != nil {
		return "", "", 0, errors.Wrapf(err, ErrMSGUnableToCreateRequestBody,
			context)
	}
	rWrapper.HTTPRequest().SetBasicAuth(m.clientID, m.clientSec)
	rWrapper.SetHeader(client.HTTPContentType, client.ContentTypeURLEncoded)
	var resBody Resp
	if err := client.Invoke(context, rWrapper, &resBody, http.StatusOK); err != nil {
		return "", "", 0, err
	}
	return resBody.AccessToken, resBody.RefreshToken, resBody.ExpiresIn, nil
}

// dynamicClientReg method gets the Client ID and Client Secret using the given Dynamic client registration request.
func (m *PasswordRefreshTokenGrantManager) dynamicClientReg(reqBody *DynamicClientRegReq) (string, string, error) {
	r, err := client.BodyReader(reqBody)
	if err != nil {
		return "", "", errors.Wrapf(err, ErrMSGUnableToParseRequestBody, DynamicClientRegMSG)
	}
	u, err := utils.ConstructURL(m.DynamicClientEndpoint, DynamicClientContext)
	if err != nil {
		return "", "", errors.Wrap(err, "cannot construct, Dynamic client registration endpoint")
	}
	rWrapper, err := client.ToHTTPRequestWrapper(http.MethodPost, u, r)
	if err != nil {
		return "", "", errors.Wrapf(err, ErrMSGUnableToCreateRequestBody, DynamicClientRegMSG)
	}
	rWrapper.HTTPRequest().SetBasicAuth(m.UserName, m.Password)
	rWrapper.SetHeader(client.HTTPContentType, client.ContentTypeApplicationJSON)

	var resBody DynamicClientRegResBody
	if err := client.Invoke(DynamicClientRegMSG, rWrapper, &resBody, http.StatusOK); err != nil {
		return "", "", err
	}
	return resBody.ClientID, resBody.ClientSecret, nil
}

// defaultClientRegBody function returns an initialized dynamic client registration request body.
func defaultClientRegBody() *DynamicClientRegReq {
	return &DynamicClientRegReq{
		CallbackURL: CallBackURL,
		ClientName:  ClientName,
		GrantType:   DynamicClientRegGrantType,
		Owner:       Owner,
		SaasApp:     true,
	}
}
