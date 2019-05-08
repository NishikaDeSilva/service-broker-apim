/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package broker

import (
	"code.cloudfoundry.org/lager"
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
	"strconv"
)

const (
	ScopeAPICreate         = "apim:api_create"
	ScopeSubscribe         = "apim:subscribe"
	ScopeAPIPublish        = "apim:api_publish"
	ErrMSGInstanceNotExist = "invalid instance Id %s"
	ErrMSGInvalidSVCPlan   = "Invalid service or Plan"

	// Logging data keys
	LogKeyAPIName         = "api-name"
	LogKeyAPIId           = "api-id"
	LogKeyServiceID       = "service-id"
	LogKeyPlanID          = "plan-id"
	LogKeyInstanceID      = "instance-id"
	LogKeyBindID          = "bind-id"
	LogKeyApplicationName = "application-name"
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

func (asb *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  serviceDetails.ServiceID,
			LogKeyPlanID:     serviceDetails.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		// Verifying whether the instance is already exists
		exists, err := isInstanceExists(instanceID)
		// Handling DB connection error
		if err != nil {
			utils.LogError("unable to get instance from DB", err, logData)
			return spec, err
		}
		if exists {
			utils.LogError(CreateAPIContext, brokerapi.ErrInstanceAlreadyExists, logData)
			return spec, brokerapi.ErrInstanceAlreadyExists
		}
		// Parse API JSON Spec
		apiParam, err := toAPIParam(serviceDetails.RawParameters)
		if err != nil {
			utils.LogError("unable to parse API parameters", err, logData)
			return spec, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
				http.StatusBadRequest, "Parsing API JSON Spec parameter")
		}

		logData.AddData(LogKeyAPIName, apiParam.APISpec.Name)

		apiID, err := asb.APIMManager.CreateAPI(&apiParam.APISpec, asb.TokenManager)
		if err != nil {
			utils.LogError("unable to create API", err, logData)
			e, ok := err.(*client.InvokeError)
			if ok && e.StatusCode == http.StatusConflict {
				return spec, brokerapi.ErrInstanceAlreadyExists
			}
			return spec, err
		}
		i := &dbutil.Instance{
			ServiceID: serviceDetails.ServiceID,
			PlanID:    serviceDetails.PlanID,
			Id:        instanceID,
			ApiID:     apiID,
			APIName:   apiParam.APISpec.Name,
		}
		logData.AddData(LogKeyAPIId, apiID)
		// Store instance in the database
		err = dbutil.StoreInstance(i)
		if err != nil {
			utils.LogError("unable to store instance", err, logData)
			errDel := asb.APIMManager.DeleteAPI(apiID, asb.TokenManager)
			if errDel != nil {
				utils.LogError("failed to cleanup, unable to delete the API", errDel, logData)
			}
			return spec, errors.Wrapf(err, "couldn't store instance in the Database instanceId: %s API ID: %s",
				instanceID, apiID)
		}
		if err = asb.APIMManager.PublishAPI(apiID, asb.TokenManager); err != nil {
			utils.LogError("unable to publish API", err, logData)
			errDel := asb.APIMManager.DeleteAPI(apiID, asb.TokenManager)
			if errDel != nil {
				utils.LogError("failed to cleanup, unable to delete the API", errDel, logData)
			}
			return spec, err
		}
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		//TODO
	} else {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return spec, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
	return spec, nil
}

func (asb *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  details.ServiceID,
			LogKeyPlanID:     details.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	if !isAPIPlan(details.ServiceID, details.PlanID) {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Deprovisioning")
	}
	instance := &dbutil.Instance{
		Id: instanceID,
	}
	exists, err := dbutil.RetrieveInstance(instance)
	if err != nil {
		utils.LogError("unable to get instance from DB", err, logData)
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	if !exists {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrInstanceDoesNotExist
	}

	logData.AddData(LogKeyAPIName, instance.APIName).AddData(LogKeyAPIId, instance.ApiID)
	err = asb.APIMManager.DeleteAPI(instance.ApiID, asb.TokenManager)
	if err != nil {
		utils.LogError("unable to delete the API", err, logData)
		return brokerapi.DeprovisionServiceSpec{}, err
	}
	err = dbutil.DeleteInstance(instance)
	if err != nil {
		utils.LogError("unable to delete the instance from the database", err, logData)
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  details.ServiceID,
			LogKeyPlanID:     details.PlanID,
			LogKeyInstanceID: instanceID,
			LogKeyBindID:     bindingID,
		},
	}
	utils.LogDebug("Binding", logData)
	if !isAPIPlan(details.ServiceID, details.PlanID) {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return brokerapi.Binding{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Binding")
	}
	// Validates the parameters before moving forward
	applicationParam, err := toApplicationParam(details.RawParameters)
	if err != nil {
		utils.LogError("invalid parameter in application json", err, logData)
		return brokerapi.Binding{}, brokerapi.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	}

	// construct the bind object
	bind := &dbutil.Bind{
		Id: bindingID,
	}
	exists, err := dbutil.RetrieveBind(bind)
	if err != nil {
		utils.LogError("unable to retrieve Bind from the database", err, logData)
		return brokerapi.Binding{}, err
	}
	// Bind exists
	if exists {
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}

	instance := &dbutil.Instance{
		Id: instanceID,
	}
	exists, err = dbutil.RetrieveInstance(instance)
	if err != nil {
		utils.LogError("unable to get instance from DB", err, logData)
		return brokerapi.Binding{}, err
	}
	if !exists {
		utils.LogError("instance does not not exists", err, logData)
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
	} else { //create service key command
		isCreateService = true
		if !utils.IsValidParams(applicationParam.AppSpec.Name) {
			return brokerapi.Binding{}, errors.New(`invalid value for "Name" parameter`)
		}
		cfAppName = applicationParam.AppSpec.Name
	}

	logData.AddData(LogKeyApplicationName, cfAppName).AddData("create-service-key command", isCreateService)
	application = &dbutil.Application{
		Name: cfAppName,
	}
	// Avoid creating new application in, Bind-service command and service-key command which failed in Generating keys
	applicationExists, err = dbutil.RetrieveApp(application)
	if err != nil {
		utils.LogError("unable to retrieve application from the database", err, logData)
		return brokerapi.Binding{}, err
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

		logData.
			AddData("throttlingTier", appCreateReq.ThrottlingTier).
			AddData("description", appCreateReq.Description).
			AddData("callbackUrl", appCreateReq.CallbackUrl)
		utils.LogDebug("creating a new application...", logData)
		appID, err := asb.APIMManager.CreateApplication(appCreateReq, asb.TokenManager)
		if err != nil {
			e, ok := err.(*client.InvokeError)
			if ok {
				logData.AddData("response code", strconv.Itoa(e.StatusCode))
			}
			utils.LogError("unable to create application", err, logData)
			return brokerapi.Binding{}, err
		}

		application.Id = appID
		// Store application before doing further API calls
		err = dbutil.StoreApp(application)
		if err != nil {
			utils.LogError("unable to store application", err, logData)
			return brokerapi.Binding{}, err
		}
		// Get the keys
		appKeyGenResp, err := asb.APIMManager.GenerateKeys(appID, asb.TokenManager)
		if err != nil {
			e, ok := err.(*client.InvokeError)
			if ok {
				logData.AddData("response code", strconv.Itoa(e.StatusCode))
			}
			utils.LogError("unable to generate keys for application", err, logData)
			return brokerapi.Binding{}, err
		}
		application.Token = appKeyGenResp.Token.AccessToken
		application.ConsumerKey = appKeyGenResp.ConsumerKey
		application.ConsumerSecret = appKeyGenResp.ConsumerSecret
		application.SubscriptionTier = applicationParam.AppSpec.SubscriptionTier
		// Update ApplicationConfig state
		err = dbutil.UpdateApp(application)
		if err != nil {
			utils.LogError("unable to store application: ", err, logData)
			return brokerapi.Binding{}, err
		}
	}
	logData.AddData(LogKeyAPIName, instance.APIName)
	if !utils.IsValidParams(applicationParam.AppSpec.SubscriptionTier) {
		nErr := errors.New(`invalid value for the SubscriptionTier "parameter"`)
		utils.LogError(`invalid value for the SubscriptionTier "parameter"`, nErr, logData)
		return brokerapi.Binding{}, nErr
	}
	utils.LogDebug("creating a subscription for application", logData)
	subscriptionID, err := asb.APIMManager.Subscribe(application.Id, instance.ApiID,
		applicationParam.AppSpec.SubscriptionTier, asb.TokenManager)
	if err != nil {
		utils.LogError("unable to create subscription", err, logData)
		return brokerapi.Binding{}, err
	}
	// Construct Bind struct and store
	bind.SubscriptionID = subscriptionID
	bind.AppName = cfAppName
	bind.InstanceID = instanceID
	bind.ServiceID = details.ServiceID
	bind.PlanID = details.PlanID
	bind.IsCreateService = isCreateService
	err = dbutil.StoreBind(bind)
	if err != nil {
		utils.LogError("unable to store bind", err, logData)
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
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  details.ServiceID,
			LogKeyPlanID:     details.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	if !isAPIPlan(details.ServiceID, details.PlanID) {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return brokerapi.UnbindSpec{}, brokerapi.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Unbinding")
	}
	// construct the bind object
	bind := &dbutil.Bind{
		Id: bindingID,
	}
	exists, err := dbutil.RetrieveBind(bind)
	if err != nil {
		utils.LogError("unable to retrieve Bind from the database", err, logData)
		return brokerapi.UnbindSpec{}, err
	}
	// Bind not exists
	if !exists {
		return brokerapi.UnbindSpec{}, brokerapi.ErrBindingDoesNotExist
	}

	logData.AddData(LogKeyApplicationName, bind.AppName)
	if bind.IsCreateService { // application created using create-service-key
		utils.LogDebug("delete service key command...", logData)
		application := &dbutil.Application{
			Name: bind.AppName,
		}
		exists, err = dbutil.RetrieveApp(application)
		if err != nil {
			utils.LogError("unable to retrieve application from the database", err, logData)
			return brokerapi.UnbindSpec{}, err
		}
		if !exists {
			utils.LogError("application does not exist", err, logData)
			return brokerapi.UnbindSpec{}, err
		}
		err = asb.APIMManager.DeleteApplication(application.Id, asb.TokenManager)
		if err != nil {
			utils.LogError("unable to delete the application", err, logData)
			return brokerapi.UnbindSpec{}, err
		}
		err = dbutil.DeleteApp(application)
		if err != nil {
			utils.LogError("unable to delete the application from the database", err, logData)
			return brokerapi.UnbindSpec{}, err
		}
	} else { // Deletes subscription
		logData.AddData("subscription-id", bind.SubscriptionID)
		err := asb.APIMManager.UnSubscribe(bind.SubscriptionID, asb.TokenManager)
		if err != nil {
			utils.LogError("unable to remove the subscription", err, logData)
			return brokerapi.UnbindSpec{}, err
		}
	}
	err = dbutil.DeleteBind(bind)
	if err != nil {
		utils.LogError("unable to delete the bind from the database", err, logData)
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
	apiPlanBindable := false
	applicationPlanBindable := true
	return []brokerapi.Service{
		{
			ID:                   constants.ServiceId,
			Name:                 constants.ServiceName,
			Description:          constants.ServiceDescription,
			Bindable:             true,
			InstancesRetrievable: false,
			PlanUpdatable:        false,
			Plans: []brokerapi.ServicePlan{
				{
					ID:          constants.APIPlanID,
					Name:        constants.APIPlanName,
					Description: constants.APIPlanDescription,
					Bindable:    &apiPlanBindable,
				},
				{
					ID:          constants.ApplicationPlanID,
					Name:        constants.ApplicationPlanName,
					Description: constants.ApplicationPlanDescription,
					Bindable:    &applicationPlanBindable,
				},
			},
		},
	}
}

func isAPIPlan(serviceID, planID string) bool {
	return (serviceID == constants.ServiceId) && (planID == constants.APIPlanID)
}

func isApplicationPlan(serviceID, planID string) bool {
	return (serviceID == constants.ServiceId) && (planID == constants.ApplicationPlanID)
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
		Id: instanceID,
	}
	exists, err := dbutil.RetrieveInstance(i)
	if err != nil {
		return false, err
	}
	return exists, nil
}
