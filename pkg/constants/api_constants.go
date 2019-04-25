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

const ErrMSGInvalidParams = "invalid parameters u: %v p: %s"
const ErrMSGUnableToParseRequestBody = "unable to parse request body: %s"
const ErrMSGUnableToCreateRequestBody = "unable to create request body: %s"
const ErrMSGUnableToParseRespBody = "unable to parse response body: %s "
const ErrMSGUnableInitiateReq = "unable to initiate request: %s"
const ErrMSGUnsuccessfulAPICall = "unsuccessful API call: %s response Code: %s URL: %s"
const ErrMSGUnableToCloseBody = "unable to close the body"
const ErrMSGAPIIDEmpty = "API ID is empty"
const ErrMSGAPPIDEmpty = "application id is empty"
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
