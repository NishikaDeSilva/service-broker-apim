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

