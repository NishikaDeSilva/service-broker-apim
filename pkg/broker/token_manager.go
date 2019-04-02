/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// BasicCredentials represents the username and password
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
	lock      sync.RWMutex
	aT        string
	rT        string
	expiresIn time.Time
}

// TokenManager is used to manage Access token & Refresh token
type TokenManager struct {
	holder map[string]*tokens
	Config *config.BrokerConfig
}

const (
	SecondSuffix                  = "s"
	ErrMSGNotEnoughArgs           = "At least one scope should be present"
	ErrMSGUnableToGetClientCreds  = "Unable to get Client credentials"
	DebugMSGClientIDClientSec     = "Client ID: %s Client Secret: %s"
	ErrMSGUnableTOGetAccessToken  = "Unable to get access token for scope: %s"
	ErrMSGUnableToParseExpireTime = "Unable parse expiresIn time"
	GenerateAccessToken           = "Generating access Token"
	DynamicClientRegMSG           = "Dynamic Client Reg"
)

var (
	once                  sync.Once
	clientID              string
	clientSec             string
	tokenEndpoint         string
	dynamicClientEndpoint string
	userName              string
	password              string
	inSecureCon           bool
)

// InitTokenManager initialize the Token Manager. This method runs only once.
// Must run before using the TokenManager
func (tm *TokenManager) InitTokenManager(scopes ...string) {
	once.Do(func() {
		if len(scopes) == 0 {
			utils.HandleErrorWithLoggerAndExit(ErrMSGNotEnoughArgs, nil)
		}
		// Initializing global vars
		userName = tm.Config.APIM.Username
		password = tm.Config.APIM.Password
		dynamicClientEndpoint = tm.Config.APIM.DynamicClientEndpoint
		tokenEndpoint = tm.Config.APIM.TokenEndpoint
		inSecureCon = tm.Config.APIM.InsecureCon

		var errDynamic error
		clientID, clientSec, errDynamic = DynamicClientReg(DefaultClientRegBody())
		if errDynamic != nil {
			utils.HandleErrorWithLoggerAndExit(ErrMSGUnableToGetClientCreds, errDynamic)
		}
		utils.LogDebug(fmt.Sprintf(DebugMSGClientIDClientSec, clientID, clientSec))

		tm.holder = make(map[string]*tokens)
		for _, scope := range scopes {
			data := accessTokenReqBody(scope)
			aT, rT, expiresIn, err := GenToken(data)
			if err != nil {
				utils.HandleErrorWithLoggerAndExit(fmt.Sprintf(ErrMSGUnableTOGetAccessToken, scope), err)
			}
			// Handling the expire time of the access token
			duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + "s")
			if err != nil {
				utils.HandleErrorWithLoggerAndExit(ErrMSGUnableToParseExpireTime, err)
			}
			tm.holder[scope] = &tokens{
				aT:        aT,
				rT:        rT,
				expiresIn: time.Now().Add(duration * time.Second),
			}
		}
	})
}

// accessTokenReqBody functions returns Token request body
func accessTokenReqBody(scope string) url.Values {
	data := url.Values{}
	data.Set(constants.UserName, userName)
	data.Add(constants.Password, password)
	data.Add(constants.GrantType, constants.GrantPassword)
	data.Add(constants.Scope, scope)
	return data
}

// refreshTokenReqBody functions returns Token request body
func refreshTokenReqBody(rT string) url.Values {
	data := url.Values{}
	data.Add(constants.RefreshToken, rT)
	data.Add(constants.GrantType, constants.GrantRefreshToken)
	return data
}

// isExpired function returns true if the difference between the current time and given time is positive
func isExpired(expiresIn time.Time) bool {
	if time.Now().Sub(expiresIn) > (0 * time.Second) {
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
	data := refreshTokenReqBody(t.rT)
	aT, rT, expiresIn, err := refreshToken(data)
	if err != nil {
		t.lock.Unlock()
		return "", err
	}
	//Parse time to type time.Duration
	duration, err := time.ParseDuration(strconv.Itoa(expiresIn) + SecondSuffix)
	tm.holder[scope] = &tokens{
		aT:        aT,
		rT:        rT,
		expiresIn: time.Now().Add(duration),
	}
	t.lock.Unlock()
	return aT, nil
}

// refreshToken function generates a new Access token and a Refresh token
func refreshToken(data url.Values) (aT, newRT string, expiresIn int, err error) {
	aT, rT, expiresIn, err := GenToken(data)
	if err != nil {
		return "", "", 0, err
	}
	return aT, rT, expiresIn, nil
}

// GenToken returns an Access token and a Refresh token from given params,
func GenToken(reqBody url.Values) (aT, rT string, expiresIn int, err error) {
	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, bytes.NewBufferString(reqBody.Encode()))
	if err != nil {
		return "", "", 0, errors.Wrapf(err, constants.ErrMSGUnableToCreateRequestBody,
			GenerateAccessToken)
	}
	req.SetBasicAuth(clientID, clientSec)
	req.Header.Add(constants.HTTPContentType, constants.ContentTypeUrlEncoded)
	var resBody TokenResp
	if err := client.Invoke(inSecureCon, GenerateAccessToken, req, &resBody, http.StatusOK); err != nil {
		return "", "", 0, err
	}
	return resBody.AccessToken, resBody.RefreshToken, resBody.ExpiresIn, nil
}

// DynamicClientReg gets the Client ID and Client Secret
func DynamicClientReg(reqBody *DynamicClientRegReqBody) (clientId, clientSecret string, er error) {
	// Encode the resBody
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(reqBody)
	if err != nil {
		return "", "", errors.Wrapf(err, constants.ErrMSGUnableToParseRequestBody, DynamicClientRegMSG)
	}

	// construct the request
	req, err := http.NewRequest(http.MethodPost, dynamicClientEndpoint, b)
	if err != nil {
		return "", "", errors.Wrapf(err, constants.ErrMSGUnableToCreateRequestBody, DynamicClientRegMSG)
	}
	req.SetBasicAuth(userName, password)
	req.Header.Set(constants.HTTPContentType, constants.ContentTypeApplicationJson)

	var resBody DynamicClientRegResBody
	if err := client.Invoke(inSecureCon, DynamicClientRegMSG, req, &resBody, http.StatusOK); err != nil {
		return "", "", err
	}
	return resBody.ClientId, resBody.ClientSecret, nil
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
