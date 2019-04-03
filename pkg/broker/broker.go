/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
)

// APIMServiceBroker struct holds the concrete implementation of the interface brokerapi.ServiceBroker
type APIMServiceBroker struct {
	BrokerConfig *config.BrokerConfig
	TokenManager *TokenManager
	APIMManager  *APIMManager
}

func (apimServiceBroker *APIMServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return Plan(), nil
}

func (apimServiceBroker *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {
	if isPlanOrg(serviceDetails) {
		exists, err := isInstanceExists(serviceDetails, instanceID)
		if err != nil {

			return spec, err
		}
		if exists {
			utils.LogError(CreateAPIContext, err)
			return spec, brokerapi.ErrInstanceAlreadyExists
		}
		apiParam, err := toApiParam(serviceDetails.RawParameters)
		if err != nil {
			return spec, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
				http.StatusBadRequest, "Parsing parameters")
		}
		s, err := apimServiceBroker.APIMManager.CreateAPI(apiParam.APISpec, apimServiceBroker.TokenManager)
		if err != nil {
			if err.(*client.InvokeError).StatusCode == http.StatusConflict {
				utils.LogError(CreateAPIContext, err)
				return spec, brokerapi.ErrInstanceAlreadyExists
			}
			return spec, err
		}
		fmt.Println(s)
	} else {
		return spec, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
	return spec, nil
}

func isInstanceExists(details brokerapi.ProvisionDetails, s string) (bool, error) {
	return false, nil
}

func (apimServiceBroker *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (apimServiceBroker *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	return brokerapi.Binding{}, nil
}

func (apimServiceBroker *APIMServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {
	return brokerapi.UnbindSpec{}, nil
}

// LastOperation ...
// If the broker provisions asynchronously, the Cloud Controller will poll this endpoint
// for the status of the provisioning operation.
func (apimServiceBroker *APIMServiceBroker) LastOperation(ctx context.Context, instanceID string,
	details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) Update(cxt context.Context, instanceID string,
	details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) GetBinding(ctx context.Context, instanceID,
bindingID string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) GetInstance(ctx context.Context,
	instanceID string) (brokerapi.GetInstanceDetailsSpec, error) {
	return brokerapi.GetInstanceDetailsSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) LastBindingOperation(ctx context.Context, instanceID,
bindingID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("not implemented")
}

// Plan returns an array of services offered by this service broker
func Plan() []brokerapi.Service {
	return []brokerapi.Service{
		{
			ID:                   constants.OrgServiceId,
			Name:                 constants.ServiceName,
			Description:          constants.ServiceDescription,
			Bindable:             constants.ServiceBindable,
			InstancesRetrievable: constants.ServiceInstancesRetrievable,
			PlanUpdatable:        constants.ServicePlanUpdateAble,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          constants.OrgPlanID,
					Name:        constants.PlanName,
					Description: constants.PlanDescription,
					Schemas: &brokerapi.ServiceSchemas{
						Instance: brokerapi.ServiceInstanceSchema{
							Create: brokerapi.Schema{
								Parameters: map[string]interface{}{
									"$schema": "http://json-schema.org/draft-04/schema#",
									"type":    "object",
									"properties": map[string]interface{}{
										"api": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"name": map[string]interface{}{
													"type": "string",
												},
												"description": map[string]interface{}{
													"type": "string",
												},
												"context": map[string]interface{}{
													"type": "string",
												},
												"version": map[string]interface{}{
													"type": "string",
												},
												"apiDefinition": map[string]interface{}{
													"type": "string",
												},
												"isDefaultVersion": map[string]interface{}{
													"type": "string",
												},
												"type": map[string]interface{}{
													"type": "string",
												},
												"transport": map[string]interface{}{
													"type": "array",
													"items": map[string]interface{}{
														"type": "string",
													},
												},
												"tiers": map[string]interface{}{
													"type": "array",
													"items": map[string]interface{}{
														"type": "string",
													},
												},
												"visibility": map[string]interface{}{
													"type": "string",
												},
												"status": map[string]interface{}{
													"type": "string",
												},
												"endpointConfig": map[string]interface{}{
													"type": "string",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func isPlanOrg(d brokerapi.ProvisionDetails) bool {
	if d.ServiceID == constants.OrgServiceId && d.PlanID == constants.OrgPlanID {
		return true
	}
	return false
}

// Parse API spec parameter
func toApiParam(params json.RawMessage) (APIParam, error) {
	var apiParam APIParam
	bytes, err := params.MarshalJSON()
	if err != nil {
		return apiParam, errors.Wrap(err, "unable to marshal raw parameter")
	}
	err = json.Unmarshal(bytes, &apiParam)
	if err != nil {
		return apiParam, errors.Wrap(err, "unable to parse API parameter")
	}
	return apiParam, nil
}
