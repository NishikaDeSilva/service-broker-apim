/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
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
	PublisherContext              = "/api/am/publisher/v0.14/apis"
	PublisherChangeAPIContext     = PublisherContext + "/change-lifecycle"
	PublishAPIContext             = "publish api"
	SubscribeContext              = "subscribe api"
	UnSubscribeContext            = "unsubscribe api"
	ApplicationDeleteContext      = "delete application"
	APIDeleteContext              = "delete API"
	APISearchContext              = "search API"
	ApplicationSearchContext      = "search Application"
)

// APIMManager handles the communication with API Manager
type APIMManager struct {
	PublisherEndpoint string
	StoreEndpoint     string
}

// CreateAPI function creates an API
func (am *APIMManager) CreateAPI(reqBody *APIReqBody, tm *TokenManager) (string, error) {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}

	aT, err := tm.Token(ScopeAPICreate)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeAPICreate)
	}

	u, err := utils.ConstructURL(am.PublisherEndpoint, PublisherContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, creation endpoint")
	}
	req, err := client.PostReq(aT, u, buf)
	if err != nil {
		return "", err
	}

	var resBody APICreateResp
	err = client.Invoke(CreateAPIContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.Id, nil
}

// Publish an API in state "Created"
func (am *APIMManager) PublishAPI(apiId string, tm *TokenManager) error {
	if apiId == "" {
		return errors.New(constants.ErrMSGAPIIDEmpty)
	}
	aT, err := tm.Token(ScopeAPIPublish)
	if err != nil {
		return errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeAPIPublish)
	}

	u, err := utils.ConstructURL(am.PublisherEndpoint, PublisherChangeAPIContext)
	if err != nil {
		return errors.Wrap(err, "cannot construct, publish API endpoint")
	}
	req, err := client.PostReq(aT, u, nil)
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Add("apiId", apiId)
	q.Add("action", "Publish")
	req.R.URL.RawQuery = q.Encode()
	err = client.Invoke(PublishAPIContext, req, nil, http.StatusOK)
	return err
}

// CreateApplication creates an application
func (am *APIMManager) CreateApplication(reqBody *ApplicationCreateReq, tm *TokenManager) (string, error) {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, StoreApplicationContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, create Application endpoint")
	}
	req, err := client.PostReq(aT, u, buf)
	if err != nil {
		return "", err
	}
	var resBody AppCreateRes
	err = client.Invoke(CreateApplicationContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.ApplicationId, nil
}

// GenerateKeys method generate keys for the given application
func (am *APIMManager) GenerateKeys(appID string, tm *TokenManager) (*ApplicationKey, error) {
	if appID == "" {
		return nil, errors.New(constants.ErrMSGAPPIDEmpty)
	}
	reqBody := defaultApplicationKeyGenerateReq()
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return nil, err
	}
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return nil, errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, GenerateApplicationKeyContext)
	if err != nil {
		return nil, errors.Wrap(err, "cannot construct, generate Application endpoint")
	}
	req, err := client.PostReq(aT, u, buf)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Add("applicationId", appID)
	req.R.URL.RawQuery = q.Encode()

	var resBody ApplicationKey
	err = client.Invoke(GenerateKeyContext, req, &resBody, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &resBody, nil
}

// Subscribe method subscribes an application to a an API
func (am *APIMManager) Subscribe(appID, apiID, tier string, tm *TokenManager) (string, error) {
	reqBody := &SubscriptionReq{
		ApplicationId: appID,
		ApiIdentifier: apiID,
		Tier:          tier,
	}
	bodyReader, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, StoreSubscriptionContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, create subscribe endpoint")
	}
	req, err := client.PostReq(aT, u, bodyReader)
	if err != nil {
		return "", err
	}
	var resBody SubscriptionResp
	err = client.Invoke(SubscribeContext, req, &resBody, http.StatusCreated)
	if err != nil {
		return "", err
	}
	return resBody.SubscriptionId, nil
}

// UnSubscribe method removes the given subscription
func (am *APIMManager) UnSubscribe(subscriptionID string, tm *TokenManager) error {
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}

	u, err := utils.ConstructURL(am.StoreEndpoint, StoreSubscriptionContext, subscriptionID)
	if err != nil {
		return errors.Wrap(err, "cannot construct, create unsubscribe endpoint")
	}
	req, err := client.DeleteReq(aT, u)
	if err != nil {
		return err
	}
	err = client.Invoke(UnSubscribeContext, req, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication method deletes the given application
func (am *APIMManager) DeleteApplication(applicationID string, tm *TokenManager) error {
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, StoreApplicationContext, applicationID)
	if err != nil {
		return errors.Wrap(err, "cannot construct, delete Application endpoint")
	}
	req, err := client.DeleteReq(aT, u)
	if err != nil {
		return err
	}
	err = client.Invoke(ApplicationDeleteContext, req, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}

// DeleteApplication method deletes the given API
func (am *APIMManager) DeleteAPI(apiID string, tm *TokenManager) error {
	aT, err := tm.Token(ScopeAPICreate)
	if err != nil {
		return errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeAPICreate)
	}
	u, err := utils.ConstructURL(am.PublisherEndpoint, PublisherContext, apiID)
	if err != nil {
		return errors.Wrap(err, "cannot construct, delete API endpoint")
	}
	req, err := client.DeleteReq(aT, u)
	if err != nil {
		return err
	}
	err = client.Invoke(APIDeleteContext, req, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}

//SearchAPI method returns API ID of the Given API
func (am *APIMManager) SearchAPI(apiName string, tm *TokenManager) (string, error) {
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.PublisherEndpoint, PublisherContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, search API endpoint")
	}
	req, err := client.GetReq(aT, u)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Add("query", apiName)
	req.R.URL.RawQuery = q.Encode()

	var resp APISearchResp
	err = client.Invoke(APISearchContext, req, &resp, http.StatusOK)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", errors.New(fmt.Sprintf("couldn't find the API %s", apiName))
	}
	if resp.Count > 1 {
		return "", errors.New(fmt.Sprintf("returned more than one API for API %s", apiName))
	}
	return resp.List[0].Id, nil
}

//SearchApplication method returns Application ID of the Given Application
func (am *APIMManager) SearchApplication(appName string, tm *TokenManager) (string, error) {
	aT, err := tm.Token(ScopeSubscribe)
	if err != nil {
		return "", errors.Wrapf(err, ErrMSGUnableToGetAccessToken, ScopeSubscribe)
	}
	u, err := utils.ConstructURL(am.StoreEndpoint,StoreApplicationContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, search Application endpoint")
	}
	req, err := client.GetReq(aT, u)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Add("query", appName)
	req.R.URL.RawQuery = q.Encode()

	var resp ApplicationSearchResp
	err = client.Invoke(ApplicationSearchContext, req, &resp, http.StatusOK)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", errors.New(fmt.Sprintf("couldn't find the Application %s", appName))
	}
	if resp.Count > 1 {
		return "", errors.New(fmt.Sprintf("returned more than one Application for %s", appName))
	}
	return resp.List[0].ApplicationId, nil
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
