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

package tokens

import (
	"github.com/jarcoal/httpmock"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	dynamicClientEndpoint     = "http://localhost"
	tokenEndpoint             = "http://localhost"
	scope                     = "scope:test"
	token                     = "token"
	refreshToken              = "refreshToken"
	expiresIn                 = 3600
	ErrMsgTestIncorrectResult = "expected value: %v but then returned value: %v"
)

var tmTest *PasswordRefreshTokenGrantManager

func init() {
	tmTest = &PasswordRefreshTokenGrantManager{
		DynamicClientEndpoint: dynamicClientEndpoint,
		UserName:              "admin",
		Password:              "admin",
		TokenEndpoint:         tokenEndpoint,
		holder: map[string]*tokens{
			scope: {
				aT: token,
				rT: refreshToken,
				// Make sure the expire time is enough to run all test cases since token
				// might be expired in the middle of the testing due to retrying.
				expiresIn: time.Now().Add(150 * time.Second),
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
			t.Errorf(ErrMsgTestIncorrectResult, expectedVal, expired)
		}
	}
}

func TestDynamicClientReg(t *testing.T) {
	t.Run("success test case", testDynamicClientRegSuccessFunc())
	t.Run("failed test case", testDynamicClientRegFailFunc())
}

func testDynamicClientRegSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, DynamicClientRegResBody{
			ClientId: "1",
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, dynamicClientEndpoint+DynamicClientContext, responder)

		clientId, _, err := tmTest.dynamicClientReg(defaultClientRegBody())
		if err != nil {
			t.Error(err)
		}
		if clientId != "1" {
			t.Errorf(ErrMsgTestIncorrectResult, "1", clientId)
		}
	}
}

func testDynamicClientRegFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusMethodNotAllowed, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, dynamicClientEndpoint+DynamicClientContext, responder)

		_, _, err = tmTest.dynamicClientReg(defaultClientRegBody())
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusMethodNotAllowed))
		}
	}
}

func TestGenToken(t *testing.T) {
	t.Run("success test case", testGenTokenSuccessFunc())
	t.Run("failure test case", testGenTokenFailFunc())
}

func testGenTokenFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusMethodNotAllowed, nil)
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, tokenEndpoint+TokenContext, responder)

		data := tmTest.accessTokenReqBody("scope:test")
		_, _, _, err = tmTest.genToken(data, GenerateAccessToken)
		if err == nil {
			t.Error("Expecting an error with code: " + strconv.Itoa(http.StatusMethodNotAllowed))
		}
	}
}

func testGenTokenSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		responder, err := httpmock.NewJsonResponder(http.StatusOK, Resp{
			AccessToken:  token,
			RefreshToken: refreshToken,
			ExpiresIn:    expiresIn,
		})
		if err != nil {
			t.Error(err)
		}
		httpmock.RegisterResponder(http.MethodPost, tokenEndpoint+TokenContext, responder)

		data := tmTest.accessTokenReqBody("scope:test")
		aT, rT, ex, err := tmTest.genToken(data, GenerateAccessToken)
		if err != nil {
			t.Error(err)
		}
		if aT != token {
			t.Errorf(ErrMsgTestIncorrectResult, token, aT)
		}
		if rT != refreshToken {
			t.Errorf(ErrMsgTestIncorrectResult, refreshToken, rT)
		}
		if ex != expiresIn {
			t.Errorf(ErrMsgTestIncorrectResult, expiresIn, ex)
		}
	}
}

func TestAccessTokenReqBody(t *testing.T) {
	data := url.Values{}
	data.Set(UserName, tmTest.UserName)
	data.Add(Password, tmTest.Password)
	data.Add(GrantType, GrantPassword)
	data.Add(Scope, scope)

	result := tmTest.accessTokenReqBody(scope)
	if !reflect.DeepEqual(result, data) {
		t.Errorf(ErrMsgTestIncorrectResult, data, result)
	}
}

func TestToken(t *testing.T) {
	testTokenSuccessFunc(t)
	testTokenRefreshFunc(t)
}

func testTokenSuccessFunc(t *testing.T) {
	aT, err := tmTest.Token(scope)
	if err != nil {
		t.Error(err)
	}
	if aT != token {
		t.Errorf(ErrMsgTestIncorrectResult, token, aT)
	}

}

func testTokenRefreshFunc(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	responder, err := httpmock.NewJsonResponder(http.StatusOK, Resp{
		AccessToken:  "newToken",
		RefreshToken: "newRefreshToken",
		ExpiresIn:    expiresIn,
	})
	if err != nil {
		t.Error(err)
	}
	tm := &PasswordRefreshTokenGrantManager{
		DynamicClientEndpoint: dynamicClientEndpoint,
		UserName:              "admin",
		Password:              "admin",
		TokenEndpoint:         tokenEndpoint,
		holder: map[string]*tokens{
			scope: {
				aT: token,
				rT: refreshToken,
				// Force fully expire the current token
				expiresIn: time.Now().Add(-10 * time.Second),
			},
		}}
	httpmock.RegisterResponder(http.MethodPost, tokenEndpoint+TokenContext, responder)

	aT, err := tm.Token(scope)
	if err != nil {
		t.Error(err)
	}
	if aT != "newToken" {
		t.Errorf(ErrMsgTestIncorrectResult, "newToken", aT)
	}
}
