/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"github.com/jarcoal/httpmock"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	dynamicClientEndpoint = "http://localhost/dynamic"
	tokenEndpoint         = "http://localhost/token"
	scope                 = "scope:test"
	token                 = "token"
	refreshToken          = "refreshToken"
	expiresIn             = 3600
)

var tm *TokenManager

func init() {
	tm = &TokenManager{
		DynamicClientEndpoint: dynamicClientEndpoint,
		UserName:              "admin",
		Password:              "admin",
		TokenEndpoint:         tokenEndpoint,
		holder: map[string]*tokens{
			scope: {
				aT:        token,
				rT:        refreshToken,
				expiresIn: time.Now().Add(10 * time.Second),
			},
		},
	}
}

func TestIsExpired(t *testing.T) {
	t.Run("not expired", testIsExpired(time.Now().Add(10*time.Second), false))
	t.Run("expired", testIsExpired(time.Now().Add((-10)*time.Second), true))
}

func testIsExpired(time time.Time, expectedVal bool) func(t *testing.T) {
	return func(t *testing.T) {
		expired := isExpired(time)
		if expired != expectedVal {
			t.Errorf(constants.ErrMsgTestIncorrectResult, expectedVal, expired)
		}
	}
}

func TestDynamicClientReg(t *testing.T) {
	t.Run("success test case", testDynamicClientRegSuccess())
	t.Run("failed test case", testDynamicClientRegFail())
}

func testDynamicClientRegSuccess() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, DynamicClientRegResBody{
			ClientId: "1",
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, dynamicClientEndpoint, responder)

		clientId, _, err := tm.DynamicClientReg(DefaultClientRegBody())
		if err != nil {
			t.Error(err)
		}
		if clientId != "1" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "1", clientId)
		}
	}
}

func testDynamicClientRegFail() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusMethodNotAllowed, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, dynamicClientEndpoint, responder)

		_, _, err = tm.DynamicClientReg(DefaultClientRegBody())
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusMethodNotAllowed))
		}
	}
}

func TestGenToken(t *testing.T) {
	t.Run("success test case", testGenTokenSuccess())
	t.Run("failure test case", testGenTokenFail())
}

func testGenTokenFail() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusMethodNotAllowed, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, tokenEndpoint, responder)

		data := tm.accessTokenReqBody("scope:test")
		_, _, _, err = tm.genToken(data, GenerateAccessToken)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusMethodNotAllowed))
		}
	}
}
func testGenTokenSuccess() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, TokenResp{
			AccessToken:  token,
			RefreshToken: refreshToken,
			ExpiresIn:    expiresIn,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, tokenEndpoint, responder)

		data := tm.accessTokenReqBody("scope:test")
		aT, rT, ex, err := tm.genToken(data, GenerateAccessToken)
		if err != nil {
			t.Error(err)
		}
		if aT != token {
			t.Errorf(constants.ErrMsgTestIncorrectResult, token, aT)
		}
		if rT != refreshToken {
			t.Errorf(constants.ErrMsgTestIncorrectResult, refreshToken, rT)
		}
		if ex != expiresIn {
			t.Errorf(constants.ErrMsgTestIncorrectResult, expiresIn, ex)
		}
	}
}

func TestAccessTokenReqBody(t *testing.T) {
	data := url.Values{}
	data.Set(constants.UserName, tm.UserName)
	data.Add(constants.Password, tm.Password)
	data.Add(constants.GrantType, constants.GrantPassword)
	data.Add(constants.Scope, scope)

	result := tm.accessTokenReqBody(scope)
	if !reflect.DeepEqual(result, data) {
		t.Errorf(constants.ErrMsgTestIncorrectResult, data, result)
	}
}

func testTokenSuccess() func(t *testing.T) {
	return func(t *testing.T) {
		aT, err := tm.Token(scope)
		if err != nil {
			t.Error(err)
		}
		if aT != token {
			t.Errorf(constants.ErrMsgTestIncorrectResult, token, aT)
		}
	}
}

func testTokenRefresh() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, TokenResp{
			AccessToken:  "newToken",
			RefreshToken: "newRefreshToken",
			ExpiresIn:    expiresIn,
		})
		if err != nil {
			t.Error(err)
		}
		// Force fully expire the current token
		tm.holder[scope].expiresIn = time.Now().Add(-10 * time.Second)
		httpmock.RegisterResponder(http.MethodPost, tokenEndpoint, responder)

		aT, err := tm.Token(scope)
		if err != nil {
			t.Error(err)
		}
		if aT != "newToken" {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "newToken", aT)
		}
	}
}

func TestToken(t *testing.T) {
	t.Run("success test case", testTokenSuccess())
	t.Run("refresh test case", testTokenRefresh())
}
