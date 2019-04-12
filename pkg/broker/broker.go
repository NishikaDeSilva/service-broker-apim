/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package broker

import (
	"context"
	"encoding/json"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/client"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
	"github.com/wso2/service-broker-apim/pkg/dbutil"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
)

const (
	ScopeAPICreate = "apim:api_create"
	ScopeAPPCreate = "apim:subscribe"
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
		// Verifying whether the instance is already exists
		exists, err := isInstanceExists(instanceID)
		// Handling DB connection error
		if err != nil {
			return spec, err
		}
		if exists {
			utils.LogError(CreateAPIContext, err)
			return spec, brokerapi.ErrInstanceAlreadyExists
		}
		// Parse API JSON Spec
		apiParam, err := toApiParam(serviceDetails.RawParameters)
		if err != nil {
			return spec, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
				http.StatusBadRequest, "Parsing API JSON Spec parameter")
		}
		apimID, err := apimServiceBroker.APIMManager.CreateAPI(apiParam.APISpec, apimServiceBroker.TokenManager)
		if err != nil {
			if err.(*client.InvokeError).StatusCode == http.StatusConflict {
				utils.LogError(CreateAPIContext, err)
				return spec, brokerapi.ErrInstanceAlreadyExists
			}
			return spec, err
		}
		i := &dbutil.Instance{
			ServiceID:  serviceDetails.ServiceID,
			PlanID:     serviceDetails.PlanID,
			InstanceID: instanceID,
			ApimID:     apimID,
		}
		// Store instance in the database
		err = dbutil.Store(i, dbutil.TableInstance)
		if err != nil {
			return spec, errors.Wrapf(err, "couldn't store instance in the Database instanceId: %s APIM ID: %s",
				instanceID, apimID)
		}
	} else {
		return spec, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
	return spec, nil
}

func (apimServiceBroker *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (apimServiceBroker *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {

	bind := &dbutil.Bind{
		BindID:     bindingID,
		InstanceID: instanceID,
	}
	exists, err := dbutil.Retrieve(&bind)
	if err != nil {
		return brokerapi.Binding{}, err
	}
	// Bind exists but parameters are not equal
	if exists {
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}
	appParam, err := toSubscriptionParam(details.RawParameters)
	if err!=nil{
		return brokerapi.Binding{}, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "Parsing Application JSON Spec parameter")
	}
	// Check for previous applications
	bind = &dbutil.Bind{
		BindID:  bindingID,
		AppName: appParam.Name,
	}
	exists, err = dbutil.Retrieve(&bind)
	if err != nil {
		return brokerapi.Binding{}, err
	}
	// Get credentials from the existing application
	if exists {

	} else { // Creates an new Application

	}

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
	return (d.ServiceID == constants.OrgServiceId) && (d.PlanID == constants.OrgPlanID)
}

// toSubscriptionParam parses subscription parameter
func toSubscriptionParam(params json.RawMessage) (ApplicationParam, error) {
	var applicationParam ApplicationParam
	err := json.Unmarshal(params, &applicationParam)
	if err != nil {
		return applicationParam, errors.Wrap(err, "unable to parse API parameter")
	}
	return applicationParam, nil
}

// toApiParam parses API spec parameter
func toApiParam(params json.RawMessage) (APIParam, error) {
	var apiParam APIParam
	err := json.Unmarshal(params, &apiParam)
	if err != nil {
		return apiParam, errors.Wrap(err, "unable to parse API parameter")
	}
	return apiParam, nil
}

// isBindExists returns true if the given bindID already exists in the DB
func isBindExists(bindID string) (bool, error) {
	i := &dbutil.Bind{
		BindID: bindID,
	}
	exists, err := dbutil.Retrieve(i)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// isInstanceExists returns true if the given instanceID already exists in the DB
func isInstanceExists(instanceID string) (bool, error) {
	i := &dbutil.Instance{
		InstanceID: instanceID,
	}
	exists, err := dbutil.Retrieve(i)
	if err != nil {
		return false, err
	}
	return exists, nil
}
