/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package broker

import (
	"context"
	"errors"
	"github.com/pivotal-cf/brokerapi"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/constants"
)

// APIMServiceBroker struct holds the concrete implementation of the interface brokerapi.ServiceBroker
type APIMServiceBroker struct {
	BrokerConfig *config.BrokerConfig
}

func (apimServiceBroker *APIMServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return plan(), nil
}

func (apimServiceBroker *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {

	return spec, nil
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

// plan returns an array of services offered by this service broker
func plan() []brokerapi.Service {
	return []brokerapi.Service{
		{
			ID:                   constants.ServiceId,
			Name:                 constants.ServiceName,
			Description:          constants.ServiceDescription,
			Bindable:             constants.ServiceBindable,
			InstancesRetrievable: constants.ServiceInstancesRetrievable,
			PlanUpdatable:        constants.ServicePlanUpdateAble,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          constants.PlanID,
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
