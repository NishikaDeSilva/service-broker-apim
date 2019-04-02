/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package constants

// CallBackUrl is a dummy value
const CallBackUrl = "www.dummy.com"

// ClientName for dynamic client registration
const ClientName = "rest_api_publisher"

// DynamicClientRegGrantType for dynamic client registration
const DynamicClientRegGrantType = "password refresh_token"

// Owner for dynamic client registration
const Owner = "admin"

const ErrMSGInvalidParams = "Invalid parameters u: %v p: %s"
const ErrMSGUnableToParseRequestBody = "Unable to parse request body: %s"
const ErrMSGUnableToCreateRequestBody = "Unable to create request body: %s"
const ErrMSGUnableToParseRespBody = "Unable to parse response body: %s "
const ErrMSGUnableInitiateReq = "Unable to initiate request: %s"
const ErrMSGUnsuccessfulAPICall = "Unsuccessful API call with response Code: %s"
const ErrMSGUnableToCloseBody = "Unable to close the body"

const HTTPContentType = "Content-Type"
const ContentTypeApplicationJson = "application/json"
const ContentTypeUrlEncoded = "application/x-www-form-urlencoded; param=value"

const UserName = "username"
const Password = "password"
const GrantPassword = "password"
const GrantRefreshToken = "refresh_token"
const GrantType = "grant_type"
const Scope = "scope"
const RefreshToken = "refresh_token"
