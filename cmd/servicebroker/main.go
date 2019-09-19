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
	"context"
	"crypto/tls"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/pkg/apim"
	"github.com/wso2/service-broker-apim/pkg/broker"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/db"
	"github.com/wso2/service-broker-apim/pkg/log"
	"github.com/wso2/service-broker-apim/pkg/tokens"
	"net/http"
	"os"
	"os/signal"
)

const (
	ErrMSGUnableToStartServerTLS = "unable to start the server on Host: %s port: %s TLS key: %s TLS cert: %s"
	ErrMSGUnableToStartServer    = "unable to start the server on Host: %s port: %s"
	InfoMSGShutdownBroker        = "starting APIM Service Broker shutdown"
	InfoMSGServerStart           = "starting APIM broker"
)

func main() {

	// Initialize the configuration
	conf, err := config.LoadConfig()
	if err != nil {
		log.HandleErrorAndExit(err)
	}
	// Initialize the logs
	logger, err := log.InitLogger(conf.Log.FilePath, conf.Log.Level)
	if err != nil {
		log.HandleErrorAndExit(err)
	}
	// Initialize HTTP client
	client.SetupClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: conf.APIM.InsecureCon},
		},
	})

	// Initialize ORM
	db.InitDB(&conf.DB)
	defer db.CloseDBCon()
	// Create tables
	db.CreateTables()

	// Initialize Token PasswordRefreshTokenGrantManager
	tManager := &tokens.PasswordRefreshTokenGrantManager{
		TokenEndpoint:         conf.APIM.TokenEndpoint,
		DynamicClientEndpoint: conf.APIM.DynamicClientEndpoint,
		UserName:              conf.APIM.Username,
		Password:              conf.APIM.Password,
	}
	tManager.InitTokenManager(tokens.ScopeAPICreate, tokens.ScopeSubscribe, tokens.ScopeAPIPublish, tokens.ScopeAPIView)

	// Initialize APIM PasswordRefreshTokenGrantManager
	apimManager := &apim.Client{
		PublisherEndpoint: conf.APIM.PublisherEndpoint,
		StoreEndpoint:     conf.APIM.StoreEndpoint,
		TokenManager:      tManager,
	}

	brokerCreds := brokerapi.BrokerCredentials{
		Username: conf.HTTP.Auth.Username,
		Password: conf.HTTP.Auth.Password,
	}
	apimServiceBroker := &broker.APIM{
		BrokerConfig: conf,
		APIMClient:   apimManager,
	}
	brokerAPI := brokerapi.New(apimServiceBroker, logger, brokerCreds)

	host := conf.HTTP.Host
	port := conf.HTTP.Port
	ld := log.NewData().
		Add("host", host).
		Add("port", port)

	server:= http.Server{
		Handler:brokerAPI,
		Addr: host + ":" + port,
	}

	// Handling terminating signal
	idleConsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, os.Kill)
		<-sigint
		log.Info(InfoMSGShutdownBroker, nil)
		if err := server.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Error("HTTP server Shutdown: %v", err, ld)
		}
		close(idleConsClosed)
	}()

	log.Info(InfoMSGServerStart, ld)
	if !conf.HTTP.TLS.Enabled {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.HandleErrorWithLoggerAndExit(
				fmt.Sprintf(ErrMSGUnableToStartServer, host, port), err)
		}
	} else {
		if err := server.ListenAndServeTLS(conf.HTTP.TLS.Cert, conf.HTTP.TLS.Key); err != http.ErrServerClosed {
			log.HandleErrorWithLoggerAndExit(fmt.Sprintf(ErrMSGUnableToStartServerTLS,
				host,
				port,
				conf.HTTP.TLS.Key,
				conf.HTTP.TLS.Cert),
				err)
		}
	}
	<-idleConsClosed
}
