/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package client

import (
	"bytes"
	"encoding/json"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"net/http"
	"testing"
)

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

func TestGenReq(t *testing.T) {
	const Host = "abc"
	const Token = "abc-w122"
	const BodyKey = "testKey"
	const BodyVal = "testVal"
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(map[string]interface{}{
		BodyKey: BodyVal,
	})
	if err != nil {
		t.Error(err)
	}
	req, err := GenReq(http.MethodDelete, Token, "http://"+Host, buf)
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
	var val map[string]interface{}
	err = json.NewDecoder(req.Body).Decode(&val)
	if err != nil {
		t.Error(err)
	}
	if val[BodyKey] != BodyVal {
		t.Errorf(constants.ErrMsgTestIncorrectResult, BodyVal, val[BodyKey])
	}
}
