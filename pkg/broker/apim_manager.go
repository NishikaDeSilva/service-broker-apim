/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"net/http"
	"net/url"
)

const (
	CreateAPIContext              = "create API"
	CreateApplicationContext      = "create application"
	GenerateKeyContext            = "Generate application keys"
	StoreApplicationContext       = "/api/am/store/v0.14/applications"
	StoreSubscriptionContext      = "/api/am/store/v0.14/subscriptions"
	GenerateApplicationKeyContext = StoreApplicationContext + "/generate-keys"
)

// APIMManager handles the communication with API Manager
type APIMManager struct {
	PublisherEndpoint string
	StoreEndpoint     string
	InsecureCon       bool
}

// CreateAPI function creates an API
func (am *APIMManager) CreateAPI(reqBody *APIReqBody, tm *TokenManager) (string, error) {
	buf, err := client.ByteBuf(reqBody)
	if err != nil {
		return "", err
	}

	aT, err := tm.Token(ScopeAPICreate)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableTOGetAccessToken, ScopeAPICreate)
	}

	req, err := client.GenReq(http.MethodPost, aT, am.PublisherEndpoint, buf)
	if err != nil {
		return "", err
	}

	var resBody APIResp
	err = client.Invoke(am.InsecureCon, CreateAPIContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.Id, nil
}

// CreateApplication creates an application
func (am *APIMManager) CreateApplication(reqBody *ApplicationCreateReq, tm *TokenManager) (string, error) {
	buf, err := client.ByteBuf(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := tm.Token(ScopeAPPCreate)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableTOGetAccessToken, ScopeAPPCreate)
	}
	req, err := client.GenReq(http.MethodPost, aT, am.StoreEndpoint+StoreApplicationContext, buf)
	if err != nil {
		return "", err
	}
	var resBody AppRes
	err = client.Invoke(am.InsecureCon, CreateApplicationContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.ApplicationId, nil
}

// GenerateKeys generate keys for the given application
func (am *APIMManager) GenerateKeys(appID string, tm *TokenManager) (*ApplicationKey, error) {
	reqBody := defaultApplicationKeyGenerateReq()
	buf, err := client.ByteBuf(reqBody)
	if err != nil {
		return nil, err
	}
	aT, err := tm.Token(ScopeAPPCreate)
	if err != nil {
		return nil, errors.Wrapf(err, ErrMSGUnableTOGetAccessToken, ScopeAPPCreate)
	}
	req, err := client.GenReq(http.MethodPost, aT, am.StoreEndpoint+GenerateApplicationKeyContext, buf)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("applicationId", appID)
	req.URL.RawQuery = q.Encode()

	var resBody ApplicationKey
	err = client.Invoke(am.InsecureCon, GenerateKeyContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	return &resBody, nil
}

func (am *APIMManager) Subscribe(appID, apiID, tier string, tm *TokenManager) (string, error) {
	reqBody := &SubscriptionReq{
		ApplicationId: appID,
		ApiIdentifier: apiID,
		Tier:          tier,
	}
	buf, err := client.ByteBuf(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := tm.Token(ScopeAPPCreate)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableTOGetAccessToken, ScopeAPPCreate)
	}
	req, err := client.GenReq(http.MethodPost, aT, am.StoreEndpoint+StoreSubscriptionContext, buf)
	if err != nil {
		return "", err
	}
	var resBody SubscriptionResp
	err = client.Invoke(am.InsecureCon, GenerateKeyContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.SubscriptionId, nil
}

func defaultApplicationKeyGenerateReq() *ApplicationKeyGenerateRequest {
	return &ApplicationKeyGenerateRequest{
		ValidityTime:       "3600",
		KeyType:            "PRODUCTION",
		AccessAllowDomains: []string{"ALL"},
		Scopes:             []string{"am_application_scope", "default"},
		SupportedGrantTypes: []string{"urn:ietf:params:oauth:grant-type:saml2-bearer", "iwa:ntlm", "refresh_token",
			"client_credentials", "password"},
	}
}
