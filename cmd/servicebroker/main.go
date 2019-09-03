/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// This is the main pkg
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
	logger, err := utils.InitLogger(brokerConfig.Log.LogFile, brokerConfig.Log.LogLevel)
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
	tManager.InitTokenManager(broker.ScopeAPICreate, broker.ScopeSubscribe, broker.ScopeAPIPublish)

	// Initialize APIM Manager
	apimManager := &broker.APIMManager{
		PublisherEndpoint: brokerConfig.APIM.PublisherEndpoint,
		StoreEndpoint:     brokerConfig.APIM.StoreEndpoint,
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
		TokenManager: tManager,
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
