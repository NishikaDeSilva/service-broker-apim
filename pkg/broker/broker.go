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
	"github.com/wso2/service-broker-apim/pkg/dbutil"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"net/http"
)

const (
	ScopeAPICreate         = "apim:api_create"
	ScopeSubscribe         = "apim:subscribe"
	ScopeAPIPublish        = "apim:api_publish"
	ErrMSGInstanceNotExist = "invalid instance Id: %s"
)

// APIMServiceBroker struct holds the concrete implementation of the interface brokerapi.ServiceBroker
type APIMServiceBroker struct {
	BrokerConfig *config.BrokerConfig
	TokenManager *TokenManager
	APIMManager  *APIMManager
}

func (asb *APIMServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return Plan(), nil
}

func (apimServiceBroker *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {
	if isPlanOrg(serviceDetails) {
		// Verifying whether the instance is already exists
		exists, err := isInstanceExists(instanceID)
		// Handling DB connection error
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to get instance: %s from database", instanceID), err)
			return spec, err
		}
		if exists {
			utils.LogError(CreateAPIContext, err)
			return spec, brokerapi.ErrInstanceAlreadyExists
		}
		// Parse API JSON Spec
		apiParam, err := toAPIParam(serviceDetails.RawParameters)
		if err != nil {
			utils.LogError("unable to parse API parameters", err)
			return spec, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
				http.StatusBadRequest, "Parsing API JSON Spec parameter")
		}
		apiID, err := apimServiceBroker.APIMManager.CreateAPI(&apiParam.APISpec, apimServiceBroker.TokenManager)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to create API: %s", apiParam.APISpec.Name), err)
			e, ok := err.(*client.InvokeError)
			if ok && e.StatusCode == http.StatusConflict {
				return spec, brokerapi.ErrInstanceAlreadyExists
			}
			return spec, err
		}
		i := &dbutil.Instance{
			ServiceID:  serviceDetails.ServiceID,
			PlanID:     serviceDetails.PlanID,
			InstanceID: instanceID,
			ApiID:      apiID,
			APIName:    apiParam.APISpec.Name,
		}
		// Store instance in the database
		err = dbutil.StoreInstance(i)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to store instance: %s", instanceID), err)
			return spec, errors.Wrapf(err, "couldn't store instance in the Database instanceId: %s API ID: %s",
				instanceID, apiID)
		}
		if err = apimServiceBroker.APIMManager.PublishAPI(apiID, apimServiceBroker.TokenManager); err != nil {
			utils.LogError(fmt.Sprintf("unable to publish API: %s", i.APIName), err)
			return spec, err
		}
	} else {
		utils.LogError(fmt.Sprintf("invalid instanceID: %s or planID: %s", instanceID, serviceDetails.PlanID),
			err)
		return spec, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
	return spec, nil
}

func (asb *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	instance := &dbutil.Instance{
		InstanceID: instanceID,
	}
	exists, err := dbutil.RetrieveInstance(instance)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to get instance: %s from DB", instanceID), err)
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	if !exists {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrInstanceDoesNotExist
	}

	err = asb.APIMManager.DeleteAPI(instance.ApiID, asb.TokenManager)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to delete the API: %s, InstanceID: %s", instance.APIName,
			instanceID), err)
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	err = dbutil.DeleteInstance(instance)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to delete the instance: %s, API: %s from the database ",
			instance.InstanceID, instance.APIName), err)
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	utils.LogDebug(fmt.Sprintf("Instance ID: %s, Bind ID: %s", instanceID, bindingID))
	// Validates the parameters before moving forward
	applicationParam, err := toApplicationParam(details.RawParameters)
	if err != nil {
		utils.LogError("invalid parameter in application json", err)
		return brokerapi.Binding{}, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	}

	// construct the bind object
	bind := &dbutil.Bind{
		BindID: bindingID,
	}
	exists, err := dbutil.RetrieveBind(bind)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to retrieve Bind from the database. BindID: %s InstanceID: %s",
			bindingID, instanceID), err)
		return brokerapi.Binding{}, err
	}
	// Bind exists
	if exists {
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}

	instance := &dbutil.Instance{
		InstanceID: instanceID,
	}
	exists, err = dbutil.RetrieveInstance(instance)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to get instance: %s from DB", instanceID), err)
		return brokerapi.Binding{}, err
	}
	if !exists {
		utils.LogError(fmt.Sprintf(ErrMSGInstanceNotExist, instanceID), err)
		return brokerapi.Binding{}, errors.Wrapf(err, ErrMSGInstanceNotExist, instanceID)
	}

	var cfAppName string
	// application instance
	var application *dbutil.Application
	// If the operation is create-service-key then no point of checking since each time an new app is created
	var applicationExists bool
	if details.BindResource != nil && details.BindResource.AppGuid != "" {
		utils.LogDebug(details.BindResource.AppGuid)
		cfAppName = details.BindResource.AppGuid
		application = &dbutil.Application{
			AppName: cfAppName,
		}
		applicationExists, err = dbutil.RetrieveApp(application)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to retrieve application: %s from the database",
				application.AppName), err)
			return brokerapi.Binding{}, err
		}
	} else { //create service key command
		cfAppName = bindingID
		application = &dbutil.Application{
			AppName: cfAppName,
		}
		utils.LogDebug(fmt.Sprintf("create service key command. Application: %s", cfAppName))
	}

	// Creates a new application
	if !applicationExists {
		if utils.ValidateParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
			applicationParam.AppSpec.CallbackUrl) {
			return brokerapi.Binding{}, errors.New("invalid parameters")
		}
		appCreateReq := &ApplicationCreateReq{
			ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
			Description:    applicationParam.AppSpec.Description,
			Name:           cfAppName,
			CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
		}
		utils.LogDebug(fmt.Sprintf("Creating a new application: %s, ThrottlingTier: %s, "+
			"Description: %s, CallbackUrl: %s ", appCreateReq.Name, appCreateReq.ThrottlingTier,
			appCreateReq.Description, appCreateReq.CallbackUrl))
		appID, err := asb.APIMManager.CreateApplication(appCreateReq, asb.TokenManager)
		if err != nil {
			e, ok := err.(*client.InvokeError)
			var msg string
			if ok {
				msg = fmt.Sprintf("unable to create application: %s, response code: %d",
					application.AppName, e.StatusCode)
			} else {
				msg = fmt.Sprintf("unable to create application: %s", application.AppName)
			}
			utils.LogError(msg, err)
			return brokerapi.Binding{}, err
		}

		application.AppID = appID
		// Store application before doing further API calls
		err = dbutil.StoreApp(application)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to store application: %s", application.AppName), err)
			return brokerapi.Binding{}, err
		}
		// Get the keys
		appKeyGenResp, err := asb.APIMManager.GenerateKeys(appID, asb.TokenManager)
		if err != nil {
			e, ok := err.(*client.InvokeError)
			var msg string
			if ok {
				msg = fmt.Sprintf("unable to generate keys for application: %s, response code: %d",
					application.AppName, e.StatusCode)
			} else {
				msg = fmt.Sprintf("unable to generate keys for application: %s", application.AppName)
			}
			utils.LogError(msg, err)
			return brokerapi.Binding{}, err
		}
		application.Token = appKeyGenResp.Token.AccessToken
		application.ConsumerKey = appKeyGenResp.ConsumerKey
		application.ConsumerSecret = appKeyGenResp.ConsumerSecret
		application.SubscriptionTier = applicationParam.AppSpec.SubscriptionTier
		// Update ApplicationConfig state
		err = dbutil.UpdateApp(application)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to store application: %s", application.AppName), err)
			return brokerapi.Binding{}, err
		}
	}
	if utils.ValidateParams(applicationParam.AppSpec.SubscriptionTier) {
		return brokerapi.Binding{}, errors.New("invalid parameters")
	}
	utils.LogDebug(fmt.Sprintf("creating a subscription for application: %s, API: %s", cfAppName, instance.APIName))
	subscriptionID, err := asb.APIMManager.Subscribe(application.AppID, instance.ApiID,
		applicationParam.AppSpec.SubscriptionTier, asb.TokenManager)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to create subscription for application: %s API: %s",
			application.AppName, instance.APIName), err)
		return brokerapi.Binding{}, err
	}
	// Construct Bind struct & store
	bind.SubscriptionID = subscriptionID
	bind.AppName = cfAppName
	bind.InstanceID = instanceID
	bind.ServiceID = details.ServiceID
	bind.PlanID = details.PlanID
	err = dbutil.StoreBind(bind)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to store bind ID: %s", bindingID), err)
		return brokerapi.Binding{}, err
	}

	credentialsMap := credentialsMap(application)
	return brokerapi.Binding{
		Credentials: credentialsMap,
	}, nil
}

func credentialsMap(app *dbutil.Application) map[string]interface{} {
	return map[string]interface{}{
		"ConsumerKey":    app.ConsumerKey,
		"ConsumerSecret": app.ConsumerSecret,
		"AccessToken":    app.Token,
	}
}
func (asb *APIMServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {

	// construct the bind object
	bind := &dbutil.Bind{
		BindID: bindingID,
	}
	exists, err := dbutil.RetrieveBind(bind)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to retrieve Bind from the database. BindID: %s InstanceID: %s",
			bindingID, instanceID), err)
		return brokerapi.UnbindSpec{}, err
	}
	// Bind not exists
	if !exists {
		return brokerapi.UnbindSpec{}, brokerapi.ErrBindingDoesNotExist
	}

	if bind.AppName == bind.BindID { // application created using create-service-key
		application := &dbutil.Application{
			AppName: bind.AppName,
		}
		exists, err = dbutil.RetrieveApp(application)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to retrieve application: %s from the database",
				application.AppName), err)
			return brokerapi.UnbindSpec{}, err
		}
		if !exists {
			utils.LogError(fmt.Sprintf("application: %s does not exist for bindID: %s",
				application.AppName, bindingID), err)
			return brokerapi.UnbindSpec{}, err
		}
		err = asb.APIMManager.DeleteApplication(application.AppID, asb.TokenManager)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to delete the appliaction: %s, bindID: %s InstanceID: %s",
				bind.AppName, bindingID, instanceID), err)
			return brokerapi.UnbindSpec{}, err
		}
		err = dbutil.DeleteApp(application)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to delete the application: %s from the database",
				application.AppName), err)
			return brokerapi.UnbindSpec{}, err
		}
	} else { // Deletes subscription
		err := asb.APIMManager.UnSubscribe(bind.SubscriptionID, asb.TokenManager)
		if err != nil {
			utils.LogError(fmt.Sprintf("unable to remove the subscription: %s, bindID: %s InstanceID: %s",
				bind.SubscriptionID, bindingID, instanceID), err)
			return brokerapi.UnbindSpec{}, err
		}
	}
	err = dbutil.DeleteBind(bind)
	if err != nil {
		utils.LogError(fmt.Sprintf("unable to delete the bind: %s from the database", bindingID), err)
		return brokerapi.UnbindSpec{}, err
	}
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

// toApplicationParam parses application parameters
func toApplicationParam(params json.RawMessage) (ApplicationParam, error) {
	var applicationParam ApplicationParam
	err := json.Unmarshal(params, &applicationParam)
	if err != nil {
		return applicationParam, errors.Wrap(err, "unable to parse application parameters")
	}
	return applicationParam, nil
}

// toAPIParam parses API spec parameter
func toAPIParam(params json.RawMessage) (APIParam, error) {
	var apiParam APIParam
	err := json.Unmarshal(params, &apiParam)
	if err != nil {
		return apiParam, errors.Wrap(err, "unable to parse API parameters")
	}
	return apiParam, nil
}

// isInstanceExists returns true if the given instanceID already exists in the DB
func isInstanceExists(instanceID string) (bool, error) {
	i := &dbutil.Instance{
		InstanceID: instanceID,
	}
	exists, err := dbutil.RetrieveInstance(i)
	if err != nil {
		return false, err
	}
	return exists, nil
}
