/*
 *  Copyright (httpClient) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package client

import (
	"bytes"
	"encoding/json"
	"github.com/jarcoal/httpmock"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

type testVal struct {
	Id   int
	Name string
}

const (
	Host             = "localhost"
	HttpMockEndpoint = "https://" + Host + "/api"
	Context          = "testing"
	Token            = "Token"
	PayloadID        = 1
	PayloadName      = "test"
	PayloadString    = `{"id": 1, "name": "test"}`
)

var payload = testVal{
	Id:   PayloadID,
	Name: PayloadName,
}

func TestB64BasicAuth(t *testing.T) {
	_, err := B64BasicAuth("", "")
	if err == nil {
		t.Errorf("Expected error didn't occur")
	}
	re1, err := B64BasicAuth("admin", "admin")
	if err != nil {
		t.Error(err)
	}
	exp := "YWRtaW46YWRtaW4="
	if re1 != exp {
		t.Errorf(constants.ErrMsgTestIncorrectResult, exp, re1)
	}
}

func TestPostReq(t *testing.T) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		t.Error(err)
	}
	req, err := PostReq(Token, HttpMockEndpoint, buf)
	if err != nil {
		t.Error(err)
	}
	if req.Method != http.MethodPost {
		t.Errorf(constants.ErrMsgTestIncorrectResult, http.MethodPost, req.Method)
	}
	if req.Host != Host {
		t.Errorf(constants.ErrMsgTestIncorrectResult, Host, req.Host)
	}
	if req.Header.Get(HeaderAuth) != (HeaderBear + Token) {
		t.Errorf(constants.ErrMsgTestIncorrectResult, HeaderBear+Token, req.Header.Get(HeaderAuth))
	}
	var val testVal
	err = json.NewDecoder(req.Body).Decode(&val)
	if err != nil {
		t.Error(err)
	}
	if val.Id != PayloadID {
		t.Errorf(constants.ErrMsgTestIncorrectResult, PayloadID, val.Id)
	}
}

func TestDeleteReq(t *testing.T) {
	req, err := DeleteReq(Token, "http://"+Host)
	if err != nil {
		t.Error(err)
	}
	if req.Method != http.MethodDelete {
		t.Errorf(constants.ErrMsgTestIncorrectResult, http.MethodDelete, req.Method)
	}
	if req.Host != Host {
		t.Errorf(constants.ErrMsgTestIncorrectResult, Host, req.Host)
	}
	if req.Header.Get(HeaderAuth) != (HeaderBear + Token) {
		t.Errorf(constants.ErrMsgTestIncorrectResult, HeaderBear+Token, req.Header.Get(HeaderAuth))
	}
}

func TestInvoke(t *testing.T) {
	t.Run("success test case", testInvokeSuccessFunc())
	t.Run("failure test case", testInvokeFailFunc())
}

func testInvokeSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", HttpMockEndpoint,
			httpmock.NewStringResponder(200, PayloadString))

		buf, err := ByteBuf(PayloadString)
		if err != nil {
			t.Error(err)
		}

		req, err := PostReq(Token, HttpMockEndpoint, buf)
		if err != nil {
			t.Error(err)
		}
		var body testVal
		err = Invoke(Context, req, &body, http.StatusOK)
		if err != nil {
			t.Error(err)
		}
		if body.Id != PayloadID {
			t.Errorf(constants.ErrMsgTestIncorrectResult, PayloadID, body.Id)
		}
	}
}

func testInvokeFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", HttpMockEndpoint,
			httpmock.NewStringResponder(http.StatusNotFound, ""))
		req, err := PostReq(Token, HttpMockEndpoint, nil)
		if err != nil {
			t.Error(err)
		}
		err = Invoke(Context, req, nil, http.StatusOK)
		if err == nil {
			t.Errorf(constants.ErrMsgTestIncorrectResult, "error response with code: "+
				strconv.Itoa(http.StatusNotFound), "reponse code: "+strconv.Itoa(http.StatusOK))
		} else {
			if e, ok := err.(*InvokeError); ok {
				if e.StatusCode != http.StatusNotFound {
					t.Errorf(constants.ErrMsgTestIncorrectResult, "response code = "+
						strconv.Itoa(http.StatusNotFound), "reponse code = "+strconv.Itoa(http.StatusOK))
				}
			} else {
				t.Errorf(constants.ErrMsgTestIncorrectResult, "type = "+reflect.TypeOf(InvokeError{}).Name(),
					"type = "+reflect.TypeOf(err).Name())
			}

		}

	}
}

func TestByteBuf(t *testing.T) {
	buf, err := ByteBuf(payload)
	if err != nil {
		t.Error(err)
	}
	var val testVal
	err = json.NewDecoder(buf).Decode(&val)
	if err != nil {
		t.Error(err)
	}
	if val.Id != PayloadID {
		t.Errorf(constants.ErrMsgTestIncorrectResult, PayloadID, val.Id)
	}
}

func TestParseBody(t *testing.T) {
	buf, err := ByteBuf(payload)
	if err != nil {
		t.Error(err)
	}
	resp := &http.Response{
		Body: ioutil.NopCloser(buf),
	}
	var body testVal
	err = ParseBody(resp, &body)
	if err != nil {
		t.Error(err)
	}
	if body.Id != PayloadID {
		t.Errorf(constants.ErrMsgTestIncorrectResult, PayloadID, body.Id)
	}
}
