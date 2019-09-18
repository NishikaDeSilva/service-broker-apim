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

package broker

type APIMaxTps struct {
	Production int64 `json:"production,omitempty"`
	Sandbox    int64 `json:"sandbox,omitempty"`
}

type ApiEndpointSecurity struct {
	Type_    string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Sequence struct {
	Name   string `json:"name"`
	Type_  string `json:"type,omitempty"`
	Id     string `json:"id,omitempty"`
	Shared bool   `json:"shared,omitempty"`
}

type Label struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ApiBusinessInformation struct {
	BusinessOwner       string `json:"businessOwner,omitempty"`
	BusinessOwnerEmail  string `json:"businessOwnerEmail,omitempty"`
	TechnicalOwner      string `json:"technicalOwner,omitempty"`
	TechnicalOwnerEmail string `json:"technicalOwnerEmail,omitempty"`
}

// CORS configuration for the API
type ApiCorsConfiguration struct {
	CorsConfigurationEnabled      bool     `json:"corsConfigurationEnabled,omitempty"`
	AccessControlAllowOrigins     []string `json:"accessControlAllowOrigins,omitempty"`
	AccessControlAllowCredentials bool     `json:"accessControlAllowCredentials,omitempty"`
	AccessControlAllowHeaders     []string `json:"accessControlAllowHeaders,omitempty"`
	AccessControlAllowMethods     []string `json:"accessControlAllowMethods,omitempty"`
}

type APIReqBody struct {
	ID string `json:"id"`
	// Name of the API
	Name string `json:"name"`
	// A brief description about the API
	Description string `json:"description"`
	// A string that represents the context of the user's request
	Context string `json:"context"`
	// The version of the API
	Version string `json:"version"`
	// If the provider value is not given, the user invoking the API will be used as the provider.
	Provider string `json:"provider,omitempty"`
	// This describes in which status of the lifecycle the API is
	Status       string `json:"status"`
	ThumbnailUri string `json:"thumbnailUri,omitempty"`
	// Swagger definition of the API which contains details about URI templates and scopes
	ApiDefinition string `json:"apiDefinition"`
	// WSDL URL if the API is based on a WSDL endpoint
	WsdlUri                 string `json:"wsdlUri,omitempty"`
	ResponseCaching         string `json:"responseCaching,omitempty"`
	CacheTimeout            int32  `json:"cacheTimeout,omitempty"`
	DestinationStatsEnabled bool   `json:"destinationStatsEnabled,omitempty"`
	IsDefaultVersion        bool   `json:"isDefaultVersion"`
	// The transport to be set. Accepted values are HTTP, WS
	Type_ string `json:"type"`
	// Supported transports for the API (http and/or https).
	Transport []string `json:"transport"`
	// Search keywords related to the API
	Tags []string `json:"tags,omitempty"`
	// The subscription tiers selected for the particular API
	Tiers []string `json:"tiers"`
	// The policy selected for the particular API
	ApiLevelPolicy string `json:"apiLevelPolicy,omitempty"`
	// Name of the Authorization header used for invoking the API. If it is not set, Authorization header name specified in tenant or system level will be used.
	AuthorizationHeader string     `json:"authorizationHeader,omitempty"`
	MaxTps              *APIMaxTps `json:"maxTps,omitempty"`
	// The visibility level of the API. Accepts one of the following. PUBLIC, PRIVATE, RESTRICTED OR CONTROLLED.
	Visibility string `json:"visibility"`
	// The user roles that are able to access the API
	VisibleRoles     []string             `json:"visibleRoles,omitempty"`
	VisibleTenants   []string             `json:"visibleTenants,omitempty"`
	EndpointConfig   string               `json:"endpointConfig"`
	EndpointSecurity *ApiEndpointSecurity `json:"endpointSecurity,omitempty"`
	// Comma separated list of gateway environments.
	GatewayEnvironments string `json:"gatewayEnvironments,omitempty"`
	// Labels of micro-gateway environments attached to the API.
	Labels    []Label    `json:"labels,omitempty"`
	Sequences []Sequence `json:"sequences,omitempty"`
	// The subscription availability. Accepts one of the following. current_tenant, all_tenants or specific_tenants.
	SubscriptionAvailability     string   `json:"subscriptionAvailability,omitempty"`
	SubscriptionAvailableTenants []string `json:"subscriptionAvailableTenants,omitempty"`
	// Map of custom properties of API
	AdditionalProperties map[string]string `json:"additionalProperties,omitempty"`
	// Is the API is restricted to certain set of publishers or creators or is it visible to all the publishers and creators. If the accessControl restriction is none, this API can be modified by all the publishers and creators, if not it can only be viewable/modifiable by certain set of publishers and creators,  based on the restriction.
	AccessControl string `json:"accessControl,omitempty"`
	// The user roles that are able to view/modify as API publisher or creator.
	AccessControlRoles  []string                `json:"accessControlRoles,omitempty"`
	BusinessInformation *ApiBusinessInformation `json:"businessInformation,omitempty"`
	CorsConfiguration   *ApiCorsConfiguration   `json:"corsConfiguration,omitempty"`
}
type APICreateResp struct {
	// UUID of the api registry artifact
	Id string `json:"id,omitempty"`
}
type ApplicationCreateReq struct {
	ThrottlingTier string `json:"throttlingTier"`
	Description    string `json:"description,omitempty"`
	Name           string `json:"name,omitempty"`
	CallbackUrl    string `json:"callbackUrl,omitempty"`
}

type APIParam struct {
	APISpec APIReqBody `json:"api"`
}

type ApplicationParam struct {
	AppSpec ApplicationCreateReq `json:"app"`
}

type SubscriptionSpec struct {
	APIName          string `json:"apiName"`
	AppName          string `json:"appName"`
	SubscriptionTier string `json:"tier"`
}

type SubscriptionParam struct {
	SubsSpec SubscriptionSpec `json:"subs"`
}

type SubscriptionReq struct {
	Tier          string `json:"tier"`
	ApiIdentifier string `json:"apiIdentifier"`
	ApplicationId string `json:"applicationId"`
}

// AppCreateReq represents the application creation request body
type AppCreateReq struct {
	ThrottlingTier string `json:"throttlingTier"`
	Description    string `json:"description"`
	Name           string `json:"name"`
	CallbackUrl    string `json:"callbackUrl"`
}

// AppCreateRes represents the application creation response body
type AppCreateRes struct {
	ApplicationId string `json:"applicationId"`
}

// ApplicationKeyGenerateRequest represents the application key generation request
type ApplicationKeyGenerateRequest struct {
	KeyType      string `json:"keyType"`
	ValidityTime string `json:"validityTime"`
	// The grant types that are supported by the application
	SupportedGrantTypes []string `json:"supportedGrantTypes,omitempty"`
	// Callback URL
	CallbackUrl string `json:"callbackUrl,omitempty"`
	// Allowed domains for the access token
	AccessAllowDomains []string `json:"accessAllowDomains"`
	// Allowed scopes for the access token
	Scopes []string `json:"scopes,omitempty"`
	// Client ID for generating access token.
	ClientId string `json:"clientId,omitempty"`
	// Client secret for generating access token. This is given together with the client Id.
	ClientSecret string `json:"clientSecret,omitempty"`
}

type ApplicationKey struct {
	// The consumer key associated with the application and identifying the client
	ConsumerKey string `json:"consumerKey,omitempty"`
	// The client secret that is used to authenticate the client with the authentication server
	ConsumerSecret string `json:"consumerSecret,omitempty"`
	// The grant types that are supported by the application
	SupportedGrantTypes []string `json:"supportedGrantTypes,omitempty"`
	// Callback URL
	CallbackUrl string `json:"callbackUrl,omitempty"`
	// Describes the state of the key generation.
	KeyState string `json:"keyState,omitempty"`
	// Describes to which endpoint the key belongs
	KeyType string `json:"keyType,omitempty"`
	// ApplicationConfig group id (if any).
	GroupId string `json:"groupId,omitempty"`
	Token   *Token `json:"token,omitempty"`
}

type Token struct {
	// Access token
	AccessToken string `json:"accessToken,omitempty"`
	// Valid scopes for the access token
	TokenScopes []string `json:"tokenScopes,omitempty"`
	// Maximum validity time for the access token
	ValidityTime int64 `json:"validityTime,omitempty"`
}

type SubscriptionResp struct {
	// The UUID of the subscription
	SubscriptionId string `json:"subscriptionId,omitempty"`
	// The UUID of the application
	ApplicationId string `json:"applicationId"`
	// The unique identifier of the API.
	ApiIdentifier string `json:"apiIdentifier"`
	Tier          string `json:"tier"`
	Status        string `json:"status,omitempty"`
}

type APISearchInfo struct {
	Provider    string `json:"provider"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Context     string `json:"context"`
	Id          string `json:"id"`
}

type APISearchResp struct {
	Previous string          `json:"previous"`
	List     []APISearchInfo `json:"list"`
	Count    int             `json:"count"`
	Next     string          `json:"next"`
}

type ApplicationSearchInfo struct {
	GroupId        string `json:"groupId"`
	Subscriber     string `json:"subscriber"`
	ThrottlingTier string `json:"throttlingTier"`
	ApplicationId  string `json:"applicationId"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
}

type ApplicationSearchResp struct {
	Previous string                  `json:"previous"`
	List     []ApplicationSearchInfo `json:"list"`
	Count    int                     `json:"count"`
	Next     string                  `json:"next"`
}
