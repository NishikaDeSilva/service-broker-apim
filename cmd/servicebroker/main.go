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
 package main

import (
	"code.cloudfoundry.org/lager"
	"crypto/tls"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/pkg/broker"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/dbutil"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	// Initialize the configuration
	brokerConfig, err := config.LoadConfig()
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	// Initialize the logs
	logger, err := utils.InitLogger(brokerConfig.Log.LogFilePath, brokerConfig.Log.Level)
	if err != nil {
		utils.HandleErrorAndExit(err)
	}
	// Initialize HTTP client
	client.SetupClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: brokerConfig.APIM.InsecureCon},
		},
	})

	// Initialize ORM
	dbutil.InitDB(&brokerConfig.DB)
	// Create tables
	dbutil.CreateTables()

	// Initialize Token Manager
	tManager := &broker.TokenManager{
		TokenEndpoint:         brokerConfig.APIM.TokenEndpoint,
		DynamicClientEndpoint: brokerConfig.APIM.DynamicClientEndpoint,
		UserName:              brokerConfig.APIM.Username,
		Password:              brokerConfig.APIM.Password,
	}
	tManager.InitTokenManager(broker.ScopeAPICreate, broker.ScopeSubscribe, broker.ScopeAPIPublish, broker.ScopeAPIView)

	// Initialize APIM Manager
	apimManager := &broker.APIMClient{
		PublisherEndpoint: brokerConfig.APIM.PublisherEndpoint,
		StoreEndpoint:     brokerConfig.APIM.StoreEndpoint,
		TokenManager:      tManager,
	}

	// Handling terminating signal
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGTERM)
	go func() {
		<-sigChannel
		utils.LogInfo(constants.InfoMSGShutdownBroker, &utils.LogData{})
		os.Exit(0)
	}()

	brokerCreds := brokerapi.BrokerCredentials{
		Username: brokerConfig.HTTP.Auth.Username,
		Password: brokerConfig.HTTP.Auth.Password,
	}
	apimServiceBroker := &broker.APIMServiceBroker{
		BrokerConfig: brokerConfig,
		APIMManager:  apimManager,
	}
	brokerAPI := brokerapi.New(apimServiceBroker, logger, brokerCreds)
	// Register router with handlers
	http.Handle("/", brokerAPI)

	host := brokerConfig.HTTP.Host
	port := brokerConfig.HTTP.Port
	utils.LogInfo(constants.InfoMSGServerStart, &utils.LogData{
		Data: lager.Data{
			"host": host,
			"port": port,
		},
	})
	if !brokerConfig.HTTP.TLS.Enabled {
		if err := http.ListenAndServe(host+":"+port, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(
				fmt.Sprintf(constants.ErrMSGUnableToStartServer, host, port), err)
		}
	} else {
		if err := http.ListenAndServeTLS(host+":"+port,
			brokerConfig.HTTP.TLS.Cert, brokerConfig.HTTP.TLS.Key, nil); err != nil {
			utils.HandleErrorWithLoggerAndExit(fmt.Sprintf(constants.ErrMSGUnableToStartServerTLS,
				host,
				port,
				brokerConfig.HTTP.TLS.Key,
				brokerConfig.HTTP.TLS.Cert),
				err)
		}
	}
}
