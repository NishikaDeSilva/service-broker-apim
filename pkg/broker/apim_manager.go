/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

package broker

import (
	"bytes"
	"encoding/json"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"net/http"
)

var (
	apimBackend string
)


func CreateAPI(apiSpec string, tm TokenManager) (string,error){
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(apiSpec)
	req, err := http.NewRequest(http.MethodPost, "https://localhost:9443/api/am/publisher/v0.14/apis", buf )
	aT,err:=tm.Token("api_create")
	if err !=nil {
		//TODO
		return "",nil

	}
	req.Header.Add("Authorization", "Bearer "+ aT)
	req.Header.Set(constants.HTTPContentType, constants.ContentTypeApplicationJson)
	var bd interface{}
	if err:=client.Invoke(true,"create API", req, bd, http.StatusCreated);err!=nil{

	}
	return "",nil
}