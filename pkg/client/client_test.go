/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package client

import (
	"github.com/wso2/service-broker-apim/pkg/constants"
	"testing"
)

func TestB64BasicAuth(t *testing.T) {
	_, err := B64BasicAuth("", "")
	if err == nil {
		t.Errorf("Expected error didn't occur")
	}
	re1, err1 := B64BasicAuth("admin", "admin")
	if err1 != nil {
		t.Error(err)
	}
	exp := "YWRtaW46YWRtaW4="
	if re1 != exp {
		t.Errorf(constants.ErrMsgTestIncorrectResult, exp, re1)
	}
}
