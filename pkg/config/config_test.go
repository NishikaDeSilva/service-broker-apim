/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package config

import (
	"fmt"
	"github.com/spf13/viper"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"os"
	"testing"
)

const ConfigFilePath = "../../config/config.yaml"

func setUpEnv(t *testing.T) {
	err := os.Setenv(constants.ConfFileEnv, ConfigFilePath)
	if err != nil {
		t.Error(err)
	}
}

func tearDownEnv(t *testing.T) {
	err := os.Unsetenv(constants.ConfFileEnv)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadConfigFile(t *testing.T) {
	t.Run("success test case", testLoadConfigFileSuccessFunc())
	t.Run("fail test case", testLoadConfigFileFailFunc())
}

func testLoadConfigFileSuccessFunc() func(t *testing.T) {
	return func(t *testing.T) {
		setUpEnv(t)
		err := loadConfigFile()
		if err != nil {
			t.Error(err)
		}
		if viper.ConfigFileUsed() != ConfigFilePath {
			t.Errorf(constants.ErrMsgTestIncorrectResult, ConfigFilePath, viper.ConfigFileUsed())
		}
		tearDownEnv(t)
	}
}

func testLoadConfigFileFailFunc() func(t *testing.T) {
	return func(t *testing.T) {
		err := loadConfigFile()
		if err == nil {
			t.Error("Expecting an error")
		} else {
			var expectedErrorMsg = fmt.Sprintf(constants.ErrMsgNoConfFile, constants.ConfFileEnv)
			if err.Error() != expectedErrorMsg {
				t.Errorf(constants.ErrMsgTestIncorrectResult, "error msg: "+expectedErrorMsg, err.Error())
			}
		}

	}
}
