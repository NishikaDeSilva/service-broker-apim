/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package config

import (
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

