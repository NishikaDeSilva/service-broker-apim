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

 // Package apim handles the interactions with APIM
package apim

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/tokens"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
	"net/url"
)

const (
	CreateAPIContext              = "create API"
	UpdateAPIContext              = "update API"
	CreateApplicationContext      = "create application"
	CreateSubscriptionContext     = "create subscription"
	UpdateApplicationContext      = "update application"
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
	ErrMSGAPIIDEmpty              = "API ID is empty"
	ErrMSGAPPIDEmpty              = "application id is empty"
)

// Client handles the communication with API PasswordRefreshTokenGrantManager.
type Client struct {
	PublisherEndpoint string
	StoreEndpoint     string
	TokenManager      tokens.Manager
}

// CreateAPI function creates an API with the provided API spec.
// Returns the API ID and any error encountered.
func (am *Client) CreateAPI(reqBody *APIReqBody) (string, error) {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}

	aT, err := am.TokenManager.Token(tokens.ScopeAPICreate)
	if err != nil {
		return "", err
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
	return resBody.ID, nil
}

// UpdateAPI function updates an existing API under the given ID with the provided API spec.
// Returns any error encountered.
func (am *Client) UpdateAPI(id string, reqBody *APIReqBody) error {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return err
	}

	aT, err := am.TokenManager.Token(tokens.ScopeAPICreate)
	if err != nil {
		return err
	}

	u, err := utils.ConstructURL(am.PublisherEndpoint, PublisherContext, id)
	if err != nil {
		return errors.Wrap(err, "cannot construct, update endpoint")
	}
	req, err := client.PutReq(aT, u, buf)
	if err != nil {
		return err
	}
	err = client.Invoke(UpdateAPIContext, req, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}

// PublishAPI publishes an API in state "Created".
// Returns any error encountered.
func (am *Client) PublishAPI(apiID string) error {
	if apiID == "" {
		return errors.New(ErrMSGAPIIDEmpty)
	}
	aT, err := am.TokenManager.Token(tokens.ScopeAPIPublish)
	if err != nil {
		return err
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
	q.Add("apiId", apiID)
	q.Add("action", "Publish")
	req.Get().URL.RawQuery = q.Encode()
	err = client.Invoke(PublishAPIContext, req, nil, http.StatusOK)
	return err
}

// CreateApplication creates an application with provided Application spec.
// Returns the Application ID and any error encountered.
func (am *Client) CreateApplication(reqBody *ApplicationCreateReq) (string, error) {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return "", err
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
	return resBody.ApplicationID, nil
}

// UpdateApplication updates an existing Application under the given ID with the provided Application spec.
// Returns any error encountered.
func (am *Client) UpdateApplication(id string, reqBody *ApplicationCreateReq) error {
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return err
	}
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return err
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, StoreApplicationContext, id)
	if err != nil {
		return errors.Wrap(err, "cannot construct, update Application endpoint")
	}
	req, err := client.PutReq(aT, u, buf)
	if err != nil {
		return err
	}
	err = client.Invoke(UpdateApplicationContext, req, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return nil
}

// GenerateKeys generates keys for the given application.
// Returns generated keys and any error encountered.
func (am *Client) GenerateKeys(appID string) (*ApplicationKeyResp, error) {
	if appID == "" {
		return nil, errors.New(ErrMSGAPPIDEmpty)
	}
	reqBody := defaultApplicationKeyGenerateReq()
	buf, err := client.BodyReader(reqBody)
	if err != nil {
		return nil, err
	}
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return nil, err
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
	req.Get().URL.RawQuery = q.Encode()

	var resBody ApplicationKeyResp
	err = client.Invoke(GenerateKeyContext, req, &resBody, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &resBody, nil
}

// Subscribe subscribes the given application to the given API with given tier.
// Returns Subscription ID and any error encountered.
func (am *Client) Subscribe(appID, apiID, tier string) (string, error) {
	reqBody := &SubscriptionReq{
		ApplicationID: appID,
		APIIdentifier: apiID,
		Tier:          tier,
	}
	bodyReader, err := client.BodyReader(reqBody)
	if err != nil {
		return "", err
	}
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return "", err
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
	return resBody.SubscriptionID, nil
}

// UnSubscribe method removes the given subscription.
// Returns any error encountered.
func (am *Client) UnSubscribe(subscriptionID string) error {
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return err
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

// DeleteApplication method deletes the given application.
// Returns any error encountered.
func (am *Client) DeleteApplication(applicationID string) error {
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return err
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

// DeleteAPI method deletes the given API.
// Returns any error encountered.
func (am *Client) DeleteAPI(apiID string) error {
	aT, err := am.TokenManager.Token(tokens.ScopeAPICreate)
	if err != nil {
		return err
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

// SearchAPI method returns API ID of the Given API
// Returns API ID and any error encountered.
func (am *Client) SearchAPI(apiName string) (string, error) {
	aT, err := am.TokenManager.Token(tokens.ScopeAPIView)
	if err != nil {
		return "", err
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
	req.Get().URL.RawQuery = q.Encode()

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
	return resp.List[0].ID, nil
}

// SearchApplication method returns Application ID of the Given Application.
// Returns Application ID and any error encountered.
func (am *Client) SearchApplication(appName string) (string, error) {
	aT, err := am.TokenManager.Token(tokens.ScopeSubscribe)
	if err != nil {
		return "", err
	}
	u, err := utils.ConstructURL(am.StoreEndpoint, StoreApplicationContext)
	if err != nil {
		return "", errors.Wrap(err, "cannot construct, search Application endpoint")
	}
	req, err := client.GetReq(aT, u)
	if err != nil {
		return "", err
	}
	q := url.Values{}
	q.Add("query", appName)
	req.Get().URL.RawQuery = q.Encode()

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
	return resp.List[0].ApplicationID, nil
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
