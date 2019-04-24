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
	ErrMSGInvalidSVCPlan    = "Invalid service or Plan"
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
	if !isOrgFlow(serviceDetails.ServiceID, serviceDetails.PlanID) {
		utils.LogError(fmt.Sprintf("invalid instanceID: %s or planID: %s", instanceID, serviceDetails.PlanID),
			errors.New(ErrMSGInvalidSVCPlan))
		return spec, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
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
	req, err := toAPICreateReq(apiParam)
	if err != nil {
		utils.LogError("unable to create API create request", err)
		return spec, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "creating API create request")
	}
	apiID, err := apimServiceBroker.APIMManager.CreateAPI(req, apimServiceBroker.TokenManager)
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
	return spec, nil
}

func (asb *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	if !isOrgFlow(details.ServiceID, details.PlanID) {
		utils.LogError(fmt.Sprintf("invalid instanceID: %s or planID: %s", instanceID, details.PlanID),
			errors.New(ErrMSGInvalidSVCPlan))
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Deprovisioning")
	}
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
	if !isOrgFlow(details.ServiceID, details.PlanID) {
		utils.LogError(fmt.Sprintf("invalid instanceID: %s or planID: %s", instanceID, details.PlanID),
			errors.New(ErrMSGInvalidSVCPlan))
		return brokerapi.Binding{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Binding")
	}
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

	var (
		// Is the create service key flow
		isCreateService = false
		// Application name
		cfAppName string
		// Application instance
		application *dbutil.Application
		// Whether the application exists or not
		applicationExists = false
	)
	if details.BindResource != nil && details.BindResource.AppGuid != "" {
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
		// If the operation is create-service-key then no point of checking since each time an new app is created
		isCreateService = true
		if !utils.IsValidParams(applicationParam.AppSpec.Name) {
			return brokerapi.Binding{}, errors.New(`invalid value for "Name" parameter`)
		}
		cfAppName = applicationParam.AppSpec.Name
		application = &dbutil.Application{
			AppName: cfAppName,
		}
		utils.LogDebug(fmt.Sprintf("create service key command. Application: %s", cfAppName))
	}

	// Creates a new application
	if !applicationExists {
		if !utils.IsValidParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
			applicationParam.AppSpec.CallbackUrl) {
			return brokerapi.Binding{}, errors.New("Invalid parameters")
		}
		appCreateReq := &ApplicationCreateReq{
			ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
			Description:    applicationParam.AppSpec.Description,
			Name:           cfAppName,
			CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
		}
		utils.LogDebug(fmt.Sprintf("creating a new application: %s, ThrottlingTier: %s, "+
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
	if !utils.IsValidParams(applicationParam.AppSpec.SubscriptionTier) {
		return brokerapi.Binding{}, errors.New(`invalid value for the SubscriptionTier "parameter"`)
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
	bind.IsCreateService = isCreateService
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
	if !isOrgFlow(details.ServiceID, details.PlanID) {
		utils.LogError(fmt.Sprintf("invalid instanceID: %s or planID: %s", instanceID, details.PlanID),
			errors.New(ErrMSGInvalidSVCPlan))
		return brokerapi.UnbindSpec{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Unbinding")
	}
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

	if bind.IsCreateService { // application created using create-service-key
		utils.LogDebug(fmt.Sprintf("delete service key command. Application: %s", bind.AppName))
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
			Bindable:             true,
			InstancesRetrievable: false,
			PlanUpdatable:        false,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          constants.OrgPlanID,
					Name:        constants.PlanName,
					Description: constants.PlanDescription,
				},
			},
		},
	}
}

func isOrgFlow(serviceID, planID string) bool {
	return (serviceID == constants.OrgServiceId) && (planID == constants.OrgPlanID)
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

// toAPICreateReq function returns API create request body from the given broker.APIParam
func toAPICreateReq(source APIParam) (*APIReqBody, error) {
	apiDef, err := utils.RawMSGToString(&source.APISpec.ApiDefinition)
	if err != nil {
		return nil, err
	}
	endpointConfig, err := utils.RawMSGToString(&source.APISpec.EndpointConfig)
	if err != nil {
		return nil, err
	}
	req := &APIReqBody{
		Name:                         source.APISpec.Name,
		Description:                  source.APISpec.Description,
		Context:                      source.APISpec.Context,
		Version:                      source.APISpec.Version,
		Provider:                     source.APISpec.Provider,
		IsDefaultVersion:             source.APISpec.IsDefaultVersion,
		Visibility:                   source.APISpec.Visibility,
		ThumbnailUri:                 source.APISpec.ThumbnailUri,
		ApiDefinition:                apiDef,
		WsdlUri:                      source.APISpec.WsdlUri,
		ResponseCaching:              source.APISpec.ResponseCaching,
		CacheTimeout:                 source.APISpec.CacheTimeout,
		DestinationStatsEnabled:      source.APISpec.DestinationStatsEnabled,
		Status:                       source.APISpec.Status,
		Type_:                        source.APISpec.Type_,
		Transport:                    source.APISpec.Transport,
		Tiers:                        source.APISpec.Tiers,
		Tags:                         source.APISpec.Tags,
		ApiLevelPolicy:               source.APISpec.ApiLevelPolicy,
		AuthorizationHeader:          source.APISpec.AuthorizationHeader,
		MaxTps:                       source.APISpec.MaxTps,
		VisibleRoles:                 source.APISpec.VisibleRoles,
		VisibleTenants:               source.APISpec.VisibleTenants,
		EndpointSecurity:             source.APISpec.EndpointSecurity,
		GatewayEnvironments:          source.APISpec.GatewayEnvironments,
		Labels:                       source.APISpec.Labels,
		Sequences:                    source.APISpec.Sequences,
		SubscriptionAvailability:     source.APISpec.SubscriptionAvailability,
		SubscriptionAvailableTenants: source.APISpec.SubscriptionAvailableTenants,
		AdditionalProperties:         source.APISpec.AdditionalProperties,
		AccessControl:                source.APISpec.AccessControl,
		AccessControlRoles:           source.APISpec.AccessControlRoles,
		BusinessInformation:          source.APISpec.BusinessInformation,
		CorsConfiguration:            source.APISpec.CorsConfiguration,
		EndpointConfig:               endpointConfig,
	}
	return req, nil
}
