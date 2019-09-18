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

package broker

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
	SecondSuffix                    = "s"
	ErrMSGNotEnoughArgs             = "At least one scope should be present"
	ErrMSGUnableToGetClientCreds    = "Unable to get Client credentials"
	ErrMSGUnableToGetAccessToken    = "Unable to get access token for scope: %s"
	ErrMSGUnableToParseExpireTime   = "Unable parse expiresIn time"
	GenerateAccessToken             = "Generating access Token"
	DynamicClientRegMSG             = "Dynamic Client Reg"
	RefreshTokenContext             = "Refresh token"
	DynamicClientContext            = "/client-registration/v0.14/register"
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

	// CallBackUrl is a dummy value
	CallBackUrl = "www.dummy.com"

	// ClientName for dynamic client registration
	ClientName = "rest_api_publisher"

	// DynamicClientRegGrantType for dynamic client registration
	DynamicClientRegGrantType = "password refresh_token"

	// Owner for dynamic client registration
	Owner = "admin"
)

// BasicCredentials represents the username and Password
type BasicCredentials struct {
	Username string
	Password string
}

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

// tokens represent the Access token & Refresh token for a particular scope
type tokens struct {
	lock      sync.RWMutex //ensures atomic writes to following fields
	aT        string
	rT        string
	expiresIn time.Time
}

// TokenManager is used to manage Access token & Refresh token
type TokenManager struct {
	once                  sync.Once
	holder                map[string]*tokens
	clientID              string
	clientSec             string
	TokenEndpoint         string
	DynamicClientEndpoint string
	UserName              string
	Password              string
}

// InitTokenManager initialize the Token Manager. This method runs only once.
// Must run before using the TokenManager
func (tm *TokenManager) InitTokenManager(scopes ...string) {
	tm.once.Do(func() {
		if len(scopes) == 0 {
			log.HandleErrorWithLoggerAndExit(ErrMSGNotEnoughArgs, nil)
		}

		var errDynamic error
		tm.clientID, tm.clientSec, errDynamic = tm.DynamicClientReg(DefaultClientRegBody())
		if errDynamic != nil {
			log.HandleErrorWithLoggerAndExit(ErrMSGUnableToGetClientCreds, errDynamic)
		}

		tm.holder = make(map[string]*tokens)
		for _, scope := range scopes {
			data := tm.accessTokenReqBody(scope)
			aT, rT, expiresIn, err := tm.genToken(data, GenerateAccessToken)
			if err != nil {
				log.HandleErrorWithLoggerAndExit(fmt.Sprintf(ErrMSGUnableToGetAccessToken, scope), err)
			}
			// Handling the expire time of the access token
			duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + "s")
			if err != nil {
				log.HandleErrorWithLoggerAndExit(ErrMSGUnableToParseExpireTime, err)
			}
			tm.holder[scope] = &tokens{
				aT:        aT,
				rT:        rT,
				expiresIn: time.Now().Add(duration),
			}
		}
	})
}

// accessTokenReqBody functions returns access token request body
func (tm *TokenManager) accessTokenReqBody(scope string) url.Values {
	data := url.Values{}
	data.Set(UserName, tm.UserName)
	data.Add(Password, tm.Password)
	data.Add(GrantType, GrantPassword)
	data.Add(Scope, scope)
	return data
}

// refreshTokenReqBody functions returns refresh token request body
func refreshTokenReqBody(rT string) url.Values {
	data := url.Values{}
	data.Add(RefreshToken, rT)
	data.Add(GrantType, GrantRefreshToken)
	return data
}

// isExpired function returns true if the difference between the current time and given time is 10s
func isExpired(expiresIn time.Time) bool {
	if time.Now().Sub(expiresIn) > (10 * time.Second) {
		return true
	}
	return false
}

// Token method returns a valid Access token. If the Access token is invalid then it will regenerate a Access token
// with the Refresh token.
func (tm *TokenManager) Token(scope string) (string, error) {
	t := tm.holder[scope]
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
	aT, rT, expiresIn, err := tm.refreshToken(t.rT)
	if err != nil {
		t.lock.Unlock()
		return "", err
	}
	//Parse time to type time.Duration
	duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + SecondSuffix)
	ld := log.NewData().
		Add("access token", aT).
		Add("refresh token", rT).
		Add("expires in", expiresIn)
	log.Debug("token details", ld)
	tm.holder[scope] = &tokens{
		aT:        aT,
		rT:        rT,
		expiresIn: time.Now().Add(duration),
	}
	t.lock.Unlock()
	return aT, nil
}

// refreshToken function generates a new Access token and a Refresh token
func (tm *TokenManager) refreshToken(rTNow string) (aT, newRT string, expiresIn int, err error) {
	data := refreshTokenReqBody(rTNow)
	aT, rT, expiresIn, err := tm.genToken(data, RefreshTokenContext)
	if err != nil {
		return "", "", 0, err
	}
	return aT, rT, expiresIn, nil
}

// genToken returns an Access token and a Refresh token from given params,
func (tm *TokenManager) genToken(reqBody url.Values, context string) (aT, rT string, expiresIn int, err error) {
	u, err := utils.ConstructURL(tm.TokenEndpoint, TokenContext)
	if err != nil {
		return "", "", 0, errors.Wrap(err, "cannot construct, token endpoint")
	}
	req, err := client.ToRequest(http.MethodPost, u, bytes.NewReader([]byte(reqBody.Encode())))
	if err != nil {
		return "", "", 0, errors.Wrapf(err, ErrMSGUnableToCreateRequestBody,
			context)
	}
	req.Get().SetBasicAuth(tm.clientID, tm.clientSec)
	req.SetHeader(client.HTTPContentType, client.ContentTypeUrlEncoded)
	var resBody TokenResp
	if err := client.Invoke(context, req, &resBody, http.StatusOK); err != nil {
		return "", "", 0, err
	}
	return resBody.AccessToken, resBody.RefreshToken, resBody.ExpiresIn, nil
}

// DynamicClientReg gets the Client ID and Client Secret
func (tm *TokenManager) DynamicClientReg(reqBody *DynamicClientRegReqBody) (clientId, clientSecret string, er error) {
	r, err := client.BodyReader(reqBody)
	if err != nil {
		return "", "", errors.Wrapf(err, ErrMSGUnableToParseRequestBody, DynamicClientRegMSG)
	}
	u, err := utils.ConstructURL(tm.DynamicClientEndpoint, DynamicClientContext)
	if err != nil {
		return "", "", errors.Wrap(err, "cannot construct, Dynamic client registration endpoint")
	}
	// construct the request
	// Not using n=client.PostReq() method since here custom headers are added
	req, err := client.ToRequest(http.MethodPost, u, r)
	if err != nil {
		return "", "", errors.Wrapf(err, ErrMSGUnableToCreateRequestBody, DynamicClientRegMSG)
	}
	req.Get().SetBasicAuth(tm.UserName, tm.Password)
	req.SetHeader(client.HTTPContentType, client.ContentTypeApplicationJson)

	var resBody DynamicClientRegResBody
	if err := client.Invoke(DynamicClientRegMSG, req, &resBody, http.StatusOK); err != nil {
		return "", "", err
	}
	return resBody.ClientId, resBody.ClientSecret, nil
}

// DefaultClientRegBody returns a dynamic client request body with values
func DefaultClientRegBody() *DynamicClientRegReqBody {
	return &DynamicClientRegReqBody{
		CallbackUrl: CallBackUrl,
		ClientName:  ClientName,
		GrantType:   DynamicClientRegGrantType,
		Owner:       Owner,
		SaasApp:     true,
	}
}
