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
	"os"
	"testing"
)

const (
	ConfigFilePath            = "../../config/config.yaml"
	ErrMsgTestIncorrectResult = "expected value: %v but then returned value: %v"
)

func setUpEnv(key, val string, t *testing.T) {
	err := os.Setenv(key, val)
	if err != nil {
		t.Error(err)
	}
}

func tearDownEnv(key string, t *testing.T) {
	err := os.Unsetenv(FilePathEnv)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadConfigFile(t *testing.T) {
	setUpEnv(FilePathEnv, ConfigFilePath, t)
	err := loadConfigFile()
	if err != nil {
		t.Error(err)
	}
	if viper.ConfigFileUsed() != ConfigFilePath {
		t.Errorf(ErrMsgTestIncorrectResult, ConfigFilePath, viper.ConfigFileUsed())
	}
	viper.Reset()
	tearDownEnv(FilePathEnv, t)
}

func TestSetDefaultConf(t *testing.T) {
	setDefaultConf()
	result, expected := viper.GetString("log.filePath"), "server.log"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("log.level"), "info"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("http.server.auth.username"), "admin"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("http.server.auth.password"), "admin"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}

	resultB := viper.GetBool("http.server.tls.enabled")
	if resultB {
		t.Errorf(ErrMsgTestIncorrectResult, false, resultB)
	}
	result, expected = viper.GetString("http.server.tls.key"), "key.pem"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("http.server.tls.cert"), "cert.pem"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("http.server.host"), "0.0.0.0"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("http.server.port"), "8444"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	resultI, expectedI := viper.GetInt("http.client.timeout"), 30
	if expectedI != resultI {
		t.Errorf(ErrMsgTestIncorrectResult, expectedI, resultI)
	}
	resultI, expectedI = viper.GetInt("http.client.minBackOff"), 1
	if expectedI != resultI {
		t.Errorf(ErrMsgTestIncorrectResult, expectedI, resultI)
	}
	resultI, expectedI = viper.GetInt("http.client.maxBackOff"), 60
	if expectedI != resultI {
		t.Errorf(ErrMsgTestIncorrectResult, expectedI, resultI)
	}
	resultI, expectedI = viper.GetInt("http.client.maxRetries"), 3
	if expectedI != resultI {
		t.Errorf(ErrMsgTestIncorrectResult, expectedI, resultI)
	}
	resultB = viper.GetBool("http.client.insecureCon")
	if !resultB {
		t.Errorf(ErrMsgTestIncorrectResult, true, resultB)
	}

	result, expected = viper.GetString("apim.username"), "admin"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("apim.password"), "admin"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}

	result, expected = viper.GetString("apim.tokenEndpoint"), "https://localhost:8243"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("apim.dynamicClientEndpoint"), "https://localhost:9443"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("apim.publisherEndpoint"), "https://localhost:9443"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("apim.storeEndpoint"), "https://localhost:9443"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.host"), "localhost"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.port"), "3306"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.username"), "root"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.password"), "root123"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.database"), "broker"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	result, expected = viper.GetString("db.database"), "broker"
	if expected != result {
		t.Errorf(ErrMsgTestIncorrectResult, expected, result)
	}
	resultB = viper.GetBool("db.logMode")
	if resultB {
		t.Errorf(ErrMsgTestIncorrectResult, false, resultB)
	}
	resultI, expectedI = viper.GetInt("db.maxRetries"), 3
	if expectedI != resultI {
		t.Errorf(ErrMsgTestIncorrectResult, expectedI, resultI)
	}
	viper.Reset()
}

func TestEnvConf(t *testing.T) {
	setUpEnv(EnvPrefix+"_DB_DATABASE", "broker_test", t)
	c, err := LoadConfig()
	if err != nil {
		t.Error(err)
	}
	if c.DB.Database != "broker_test" {
		t.Errorf(ErrMsgTestIncorrectResult, "broker_test", c.DB.Database)
	}
	tearDownEnv(EnvPrefix+"_db_database", t)
	viper.Reset()
}
