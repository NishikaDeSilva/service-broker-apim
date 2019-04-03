/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker


type ApiMaxTps struct {
	Production int64 `json:"production,omitempty"`
	Sandbox int64 `json:"sandbox,omitempty"`
}
type ApiEndpointSecurity struct {
	Type_ string `json:"type,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"Password,omitempty"`
}

type Sequence struct {
	Name string `json:"name"`
	Type_ string `json:"type,omitempty"`
	Id string `json:"id,omitempty"`
	Shared bool `json:"shared,omitempty"`
}

type Label struct {
	Name string `json:"name"`
	Description string `json:"description,omitempty"`
}

type ApiBusinessInformation struct {
	BusinessOwner string `json:"businessOwner,omitempty"`
	BusinessOwnerEmail string `json:"businessOwnerEmail,omitempty"`
	TechnicalOwner string `json:"technicalOwner,omitempty"`
	TechnicalOwnerEmail string `json:"technicalOwnerEmail,omitempty"`
}

// CORS configuration for the API
type ApiCorsConfiguration struct {
	CorsConfigurationEnabled bool `json:"corsConfigurationEnabled,omitempty"`
	AccessControlAllowOrigins []string `json:"accessControlAllowOrigins,omitempty"`
	AccessControlAllowCredentials bool `json:"accessControlAllowCredentials,omitempty"`
	AccessControlAllowHeaders []string `json:"accessControlAllowHeaders,omitempty"`
	AccessControlAllowMethods []string `json:"accessControlAllowMethods,omitempty"`
}

type APIReqBody struct {
	//Id string `json:"id,omitempty"`
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
	Status string `json:"status"`
	ThumbnailUri string `json:"thumbnailUri,omitempty"`
	// Swagger definition of the API which contains details about URI templates and scopes
	ApiDefinition string `json:"apiDefinition"`
	// WSDL URL if the API is based on a WSDL endpoint
	WsdlUri string `json:"wsdlUri,omitempty"`
	ResponseCaching string `json:"responseCaching,omitempty"`
	CacheTimeout int32 `json:"cacheTimeout,omitempty"`
	DestinationStatsEnabled string `json:"destinationStatsEnabled,omitempty"`
	IsDefaultVersion bool `json:"isDefaultVersion"`
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
	AuthorizationHeader string `json:"authorizationHeader,omitempty"`
	MaxTps *ApiMaxTps `json:"maxTps,omitempty"`
	// The visibility level of the API. Accepts one of the following. PUBLIC, PRIVATE, RESTRICTED OR CONTROLLED.
	Visibility string `json:"visibility"`
	// The user roles that are able to access the API
	VisibleRoles []string `json:"visibleRoles,omitempty"`
	VisibleTenants []string `json:"visibleTenants,omitempty"`
	EndpointConfig string `json:"endpointConfig"`
	EndpointSecurity *ApiEndpointSecurity `json:"endpointSecurity,omitempty"`
	// Comma separated list of gateway environments.
	GatewayEnvironments string `json:"gatewayEnvironments,omitempty"`
	// Labels of micro-gateway environments attached to the API.
	Labels []Label `json:"labels,omitempty"`
	Sequences []Sequence `json:"sequences,omitempty"`
	// The subscription availability. Accepts one of the following. current_tenant, all_tenants or specific_tenants.
	SubscriptionAvailability string `json:"subscriptionAvailability,omitempty"`
	SubscriptionAvailableTenants []string `json:"subscriptionAvailableTenants,omitempty"`
	// Map of custom properties of API
	AdditionalProperties map[string]string `json:"additionalProperties,omitempty"`
	// Is the API is restricted to certain set of publishers or creators or is it visible to all the publishers and creators. If the accessControl restriction is none, this API can be modified by all the publishers and creators, if not it can only be viewable/modifiable by certain set of publishers and creators,  based on the restriction.
	AccessControl string `json:"accessControl,omitempty"`
	// The user roles that are able to view/modify as API publisher or creator.
	AccessControlRoles []string `json:"accessControlRoles,omitempty"`
	BusinessInformation *ApiBusinessInformation `json:"businessInformation,omitempty"`
	CorsConfiguration *ApiCorsConfiguration `json:"corsConfiguration,omitempty"`
}

type APIResp struct {
	// UUID of the api registry artifact
	Id string `json:"id,omitempty"`
}

type APIParam struct {
	APISpec APIReqBody `json:"api,omitempty"`
}