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

func TestLoadConfig(t *testing.T) {
	setUpEnv(t)
	config, err := LoadConfig()
	if err != nil {
		t.Error(err)
	}
	if config.HTTP.Host != defaultConf().HTTP.Host {
		t.Errorf(constants.ErrMsgTestIncorrectResult, defaultConf().HTTP.Host, config.HTTP.Host)
	}
	tearDownEnv(t)
}

func TestLoadConfigFile(t *testing.T) {
	t.Run("success test case", testLoadConfigFileSuccess())
	t.Run("failed test case", testLoadConfigFileFailed())
}

func testLoadConfigFileSuccess() func(t *testing.T) {
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

func testLoadConfigFileFailed() func(t *testing.T) {
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
