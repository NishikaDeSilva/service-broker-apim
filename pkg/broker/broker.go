/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */
package broker

import (
	"code.cloudfoundry.org/lager"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/brokerapi/domain"
	"github.com/pivotal-cf/brokerapi/domain/apiresponses"
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
	LogKeyAPIName                  = "api-name"
	LogKeyAPIID                    = "api-id"
	LogKeyAPPID                    = "app-id"
	LogKeyAPPName                  = "app-name"
	LogKeySubsID                   = "subs-id"
	LogKeyServiceID                = "service-id"
	LogKeyPlanID                   = "plan-id"
	LogKeyInstanceID               = "instance-id"
	LogKeyBindID                   = "bind-id"
	LogKeyApplicationName          = "application-name"
	ErrMSGUnableToStoreInstance    = "unable to store instance in DB"
	ErrActionUnableToStoreInstance = "store instance in DB"
)

// APIMServiceBroker struct holds the concrete implementation of the interface brokerapi.ServiceBroker
type APIMServiceBroker struct {
	BrokerConfig *config.BrokerConfig
	//TokenManager *TokenManager
	APIMManager *APIMManager
}

func (asb *APIMServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
	return Plan(), nil
}

func (asb *APIMServiceBroker) Provision(ctx context.Context, instanceID string,
	serviceDetails domain.ProvisionDetails, asyncAllowed bool) (spec domain.ProvisionedServiceSpec, err error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  serviceDetails.ServiceID,
			LogKeyPlanID:     serviceDetails.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	if isAPIPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createAPIService(instanceID, serviceDetails)
	} else if isApplicationPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createAppService(instanceID, serviceDetails)
	} else if isSubscriptionPlan(serviceDetails.ServiceID, serviceDetails.PlanID) {
		return asb.createSubscriptionService(instanceID, serviceDetails)
	} else {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return spec, apiresponses.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "provisioning")
	}
}

func (asb *APIMServiceBroker) Deprovision(ctx context.Context, instanceID string,
	details domain.DeprovisionDetails, asyncAllowed bool) (domain.DeprovisionServiceSpec, error) {
	//var logData = &utils.LogData{
	//	Data: lager.Data{
	//		LogKeyServiceID:  details.ServiceID,
	//		LogKeyPlanID:     details.PlanID,
	//		LogKeyInstanceID: instanceID,
	//	},
	//}
	//if !isAPIPlan(details.ServiceID, details.PlanID) {
	//	utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
	//	return domain.DeprovisionServiceSpec{}, domain.NewFailureResponse(errors.New("invalid Plan or Service"),
	//		http.StatusBadRequest, "Deprovisioning")
	//}
	//instance := &dbutil.Instance{
	//	Id: instanceID,
	//}
	//exists, err := dbutil.RetrieveInstance(instance)
	//if err != nil {
	//	utils.LogError("unable to get instance from DB", err, logData)
	//	return domain.DeprovisionServiceSpec{}, err
	//}
	//if !exists {
	//	return domain.DeprovisionServiceSpec{}, domain.ErrInstanceDoesNotExist
	//}
	//
	//logData.AddData(LogKeyAPIName, instance.APIName).AddData(LogKeyAPIID, instance.ApiID)
	//err = asb.APIMManager.DeleteAPI(instance.ApiID)
	//if err != nil {
	//	utils.LogError("unable to delete the API", err, logData)
	//	return domain.DeprovisionServiceSpec{}, err
	//}
	//err = dbutil.DeleteInstance(instance)
	//if err != nil {
	//	utils.LogError("unable to delete the instance from the database", err, logData)
	//	return domain.DeprovisionServiceSpec{}, err
	//}

	return domain.DeprovisionServiceSpec{}, nil
}

func (asb *APIMServiceBroker) Bind(ctx context.Context, instanceID, bindingID string,
	details domain.BindDetails, asyncAllowed bool) (domain.Binding, error) {
	//var logData = &utils.LogData{
	//	Data: lager.Data{
	//		LogKeyServiceID:  details.ServiceID,
	//		LogKeyPlanID:     details.PlanID,
	//		LogKeyInstanceID: instanceID,
	//		LogKeyBindID:     bindingID,
	//	},
	//}
	//utils.LogDebug("Binding", logData)
	//if !isAPIPlan(details.ServiceID, details.PlanID) {
	//	utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
	//	return domain.Binding{}, domain.NewFailureResponse(errors.New("invalid Plan or Service"),
	//		http.StatusBadRequest, "Binding")
	//}
	//// Validates the parameters before moving forward
	//applicationParam, err := toApplicationParam(details.RawParameters)
	//if err != nil {
	//	utils.LogError("invalid parameter in application json", err, logData)
	//	return domain.Binding{}, domain.NewFailureResponse(errors.New("invalid parameter"),
	//		http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	//}
	//
	//// construct the bind object
	//bind := &dbutil.Bind{
	//	Id: bindingID,
	//}
	//exists, err := dbutil.RetrieveBind(bind)
	//if err != nil {
	//	utils.LogError("unable to retrieve Bind from the database", err, logData)
	//	return domain.Binding{}, err
	//}
	//// Bind exists
	//if exists {
	//	return domain.Binding{}, domain.ErrBindingAlreadyExists
	//}
	//
	//instance := &dbutil.Instance{
	//	Id: instanceID,
	//}
	//exists, err = dbutil.RetrieveInstance(instance)
	//if err != nil {
	//	utils.LogError("unable to get instance from DB", err, logData)
	//	return domain.Binding{}, err
	//}
	//if !exists {
	//	utils.LogError("instance does not not exists", err, logData)
	//	return domain.Binding{}, errors.Wrapf(err, ErrMSGInstanceNotExist, instanceID)
	//}
	//
	//var (
	//	// Is the create service key flow
	//	isCreateService = false
	//	// Application name
	//	cfAppName string
	//	// Application instance
	//	application *dbutil.Application
	//	// Whether the application exists or not
	//	applicationExists = false
	//)
	//if details.BindResource != nil && details.BindResource.AppGuid != "" {
	//	cfAppName = details.BindResource.AppGuid
	//} else { //create service key command
	//	isCreateService = true
	//	if !utils.IsValidParams(applicationParam.AppSpec.Name) {
	//		return domain.Binding{}, errors.New(`invalid value for "Name" parameter`)
	//	}
	//	cfAppName = applicationParam.AppSpec.Name
	//}
	//
	//logData.AddData(LogKeyApplicationName, cfAppName).AddData("create-service-key command", isCreateService)
	//application = &dbutil.Application{
	//	Name: cfAppName,
	//}
	//// Avoid creating new application in, Bind-service command and service-key command which failed in Generating keys
	//applicationExists, err = dbutil.RetrieveApp(application)
	//if err != nil {
	//	utils.LogError("unable to retrieve application from the database", err, logData)
	//	return domain.Binding{}, err
	//}
	//// Creates a new application
	//if !applicationExists {
	//	if !utils.IsValidParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
	//		applicationParam.AppSpec.CallbackUrl) {
	//		return domain.Binding{}, errors.New("Invalid parameters")
	//	}
	//	appCreateReq := &ApplicationCreateReq{
	//		ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
	//		Description:    applicationParam.AppSpec.Description,
	//		Name:           cfAppName,
	//		CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
	//	}
	//
	//	logData.
	//		AddData("throttlingTier", appCreateReq.ThrottlingTier).
	//		AddData("description", appCreateReq.Description).
	//		AddData("callbackUrl", appCreateReq.CallbackUrl)
	//	utils.LogDebug("creating a new application...", logData)
	//	appID, err := asb.APIMManager.CreateApplication(appCreateReq)
	//	if err != nil {
	//		e, ok := err.(*client.InvokeError)
	//		if ok {
	//			logData.AddData("response code", strconv.Itoa(e.StatusCode))
	//		}
	//		utils.LogError("unable to create application", err, logData)
	//		return domain.Binding{}, err
	//	}
	//
	//	application.Id = appID
	//	// Store application before doing further API calls
	//	err = dbutil.StoreApp(application)
	//	if err != nil {
	//		utils.LogError("unable to store application", err, logData)
	//		return domain.Binding{}, err
	//	}
	//	// Get the keys
	//	appKeyGenResp, err := asb.APIMManager.GenerateKeys(appID)
	//	if err != nil {
	//		e, ok := err.(*client.InvokeError)
	//		if ok {
	//			logData.AddData("response code", strconv.Itoa(e.StatusCode))
	//		}
	//		utils.LogError("unable to generate keys for application", err, logData)
	//		return domain.Binding{}, err
	//	}
	//	application.Token = appKeyGenResp.Token.AccessToken
	//	application.ConsumerKey = appKeyGenResp.ConsumerKey
	//	application.ConsumerSecret = appKeyGenResp.ConsumerSecret
	//	application.SubscriptionTier = applicationParam.AppSpec.SubscriptionTier
	//	// Update ApplicationConfig state
	//	err = dbutil.UpdateApp(application)
	//	if err != nil {
	//		utils.LogError("unable to store application: ", err, logData)
	//		return domain.Binding{}, err
	//	}
	//}
	//logData.AddData(LogKeyAPIName, instance.APIName)
	//if !utils.IsValidParams(applicationParam.AppSpec.SubscriptionTier) {
	//	nErr := errors.New(`invalid value for the SubscriptionTier "parameter"`)
	//	utils.LogError(`invalid value for the SubscriptionTier "parameter"`, nErr, logData)
	//	return domain.Binding{}, nErr
	//}
	//utils.LogDebug("creating a subscription for application", logData)
	//subscriptionID, err := asb.APIMManager.Subscribe(application.Id, instance.ApiID,
	//	applicationParam.AppSpec.SubscriptionTier)
	//if err != nil {
	//	utils.LogError("unable to create subscription", err, logData)
	//	return domain.Binding{}, err
	//}
	//// Construct Bind struct and store
	//bind.SubscriptionID = subscriptionID
	//bind.AppName = cfAppName
	//bind.InstanceID = instanceID
	//bind.ServiceID = details.ServiceID
	//bind.PlanID = details.PlanID
	//bind.IsCreateService = isCreateService
	//err = dbutil.StoreBind(bind)
	//if err != nil {
	//	utils.LogError("unable to store bind", err, logData)
	//	return domain.Binding{}, err
	//}

	credentialsMap := credentialsMap(nil)
	return domain.Binding{
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
	details domain.UnbindDetails, asyncAllowed bool) (domain.UnbindSpec, error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  details.ServiceID,
			LogKeyPlanID:     details.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	if !isAPIPlan(details.ServiceID, details.PlanID) {
		utils.LogError("invalid instance id or plan id", errors.New(ErrMSGInvalidSVCPlan), logData)
		return domain.UnbindSpec{}, apiresponses.NewFailureResponse(errors.New("invalid Plan or Service"),
			http.StatusBadRequest, "Unbinding")
	}
	// construct the bind object
	bind := &dbutil.Bind{
		Id: bindingID,
	}
	exists, err := dbutil.RetrieveBind(bind)
	if err != nil {
		utils.LogError("unable to retrieve Bind from the database", err, logData)
		return domain.UnbindSpec{}, err
	}
	// Bind not exists
	if !exists {
		return domain.UnbindSpec{}, apiresponses.ErrBindingDoesNotExist
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
			return domain.UnbindSpec{}, err
		}
		if !exists {
			utils.LogError("application does not exist", err, logData)
			return domain.UnbindSpec{}, err
		}
		err = asb.APIMManager.DeleteApplication(application.Id)
		if err != nil {
			utils.LogError("unable to delete the application", err, logData)
			return domain.UnbindSpec{}, err
		}
		err = dbutil.DeleteApp(application)
		if err != nil {
			utils.LogError("unable to delete the application from the database", err, logData)
			return domain.UnbindSpec{}, err
		}
	} else { // Deletes subscription
		logData.AddData("subscription-id", bind.SubscriptionID)
		err := asb.APIMManager.UnSubscribe(bind.SubscriptionID)
		if err != nil {
			utils.LogError("unable to remove the subscription", err, logData)
			return domain.UnbindSpec{}, err
		}
	}
	err = dbutil.DeleteBind(bind)
	if err != nil {
		utils.LogError("unable to delete the bind from the database", err, logData)
		return domain.UnbindSpec{}, err
	}
	return domain.UnbindSpec{}, nil
}

// LastOperation ...
// If the broker provisions asynchronously, the Cloud Controller will poll this endpoint
// for the status of the provisioning operation.
func (apimServiceBroker *APIMServiceBroker) LastOperation(ctx context.Context, instanceID string,
	details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) Update(cxt context.Context, instanceID string,
	details domain.UpdateDetails, asyncAllowed bool) (domain.UpdateServiceSpec, error) {
	return domain.UpdateServiceSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) GetBinding(ctx context.Context, instanceID,
bindingID string) (domain.GetBindingSpec, error) {
	return domain.GetBindingSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) GetInstance(ctx context.Context,
	instanceID string) (domain.GetInstanceDetailsSpec, error) {
	return domain.GetInstanceDetailsSpec{}, errors.New("not implemented")
}

func (apimServiceBroker *APIMServiceBroker) LastBindingOperation(ctx context.Context, instanceID,
bindingID string, details domain.PollDetails) (domain.LastOperation, error) {
	return domain.LastOperation{}, errors.New("not implemented")
}

// Plan returns an array of services offered by this service broker
func Plan() []domain.Service {
	apiPlanBindable := false
	applicationPlanBindable := true
	subscriptionPlanBindable := false
	return []domain.Service{
		{
			ID:                   constants.ServiceId,
			Name:                 constants.ServiceName,
			Description:          constants.ServiceDescription,
			Bindable:             false,
			InstancesRetrievable: false,
			PlanUpdatable:        false,
			Plans: []domain.ServicePlan{
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
				{
					ID:          constants.SubscriptionPlanID,
					Name:        constants.SubscriptionPlanName,
					Description: constants.SubscriptionPlanDescription,
					Bindable:    &subscriptionPlanBindable,
				},
			},
		},
	}
}

func (asb *APIMServiceBroker) createAPIService(instanceID string, serviceDetails domain.ProvisionDetails, ) (spec domain.ProvisionedServiceSpec, err error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  serviceDetails.ServiceID,
			LogKeyPlanID:     serviceDetails.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	utils.LogDebug("creating API service instance", logData)
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		utils.LogError("unable to get instance from DB", err, logData)
		return spec, apiresponses.NewFailureResponse(errors.New("unable to query database"),
			http.StatusInternalServerError, "getting the instance information from DB")
	}
	if exists {
		utils.LogError(CreateAPIContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return spec, apiresponses.ErrInstanceAlreadyExists
	}
	// Parse API JSON Spec
	apiParam, err := toAPIParam(serviceDetails.RawParameters)
	if err != nil {
		utils.LogError("unable to parse API parameters", err, logData)
		return spec, apiresponses.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "Parsing API JSON Spec parameter")
	}
	logData.AddData(LogKeyAPIName, apiParam.APISpec.Name)

	apiID, err := asb.APIMManager.CreateAPI(&apiParam.APISpec)
	if err != nil {
		utils.LogError("unable to create API", err, logData)
		e, ok := err.(*client.InvokeError)
		if ok && e.StatusCode == http.StatusConflict {
			return spec, apiresponses.ErrInstanceAlreadyExists
		}
		return spec, apiresponses.NewFailureResponse(errors.New("unable to create the API"),
			http.StatusInternalServerError, "unable to create the API")
	}

	logData.AddData(LogKeyAPIID, apiID)
	utils.LogDebug("created API in APIM", logData)

	i := &dbutil.Instance{
		ServiceID:        serviceDetails.ServiceID,
		PlanID:           serviceDetails.PlanID,
		Id:               instanceID,
		APIMResourceID:   apiID,
		APIMResourceName: apiParam.APISpec.Name,
	}

	// Store instance in the database
	err = dbutil.StoreInstance(i)
	if err != nil {
		utils.LogError("unable to store instance", err, logData)
		errDel := asb.APIMManager.DeleteAPI(apiID)
		if errDel != nil {
			utils.LogError("failed to cleanup, unable to delete the API", errDel, logData)
		}
		return spec, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionUnableToStoreInstance)
	}
	if err = asb.APIMManager.PublishAPI(apiID); err != nil {
		utils.LogError("unable to publish API", err, logData)
		errDel := asb.APIMManager.DeleteAPI(apiID)
		if errDel != nil {
			utils.LogError("failed to cleanup, unable to delete the API", errDel, logData)
		}
		return spec, apiresponses.NewFailureResponse(errors.New("unable to publish the API"),
			http.StatusInternalServerError, "unable to publish the API")
	}
	utils.LogDebug("published API in APIM", logData)
	return spec, nil
}

func (asb *APIMServiceBroker) createAppService(instanceID string, serviceDetails domain.ProvisionDetails) (domain.ProvisionedServiceSpec, error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  serviceDetails.ServiceID,
			LogKeyPlanID:     serviceDetails.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		utils.LogError("unable to get instance from DB", err, logData)
		return domain.ProvisionedServiceSpec{}, err
	}
	if exists {
		utils.LogError(CreateApplicationContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}
	// Validates the parameters before moving forward
	applicationParam, err := toApplicationParam(serviceDetails.RawParameters)
	if err != nil {
		utils.LogError("invalid parameter in application json", err, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameter"),
			http.StatusBadRequest, "parsing ApplicationConfig JSON Spec parameter")
	}
	if !utils.IsValidParams(applicationParam.AppSpec.ThrottlingTier, applicationParam.AppSpec.Description,
		applicationParam.AppSpec.CallbackUrl, applicationParam.AppSpec.Name) {
		return domain.ProvisionedServiceSpec{}, apiresponses.NewFailureResponse(errors.New("invalid parameters"),
			http.StatusBadRequest, "Parsing Subscription parameters")
	}
	appCreateReq := &ApplicationCreateReq{
		ThrottlingTier: applicationParam.AppSpec.ThrottlingTier,
		Description:    applicationParam.AppSpec.Description,
		Name:           applicationParam.AppSpec.Name,
		CallbackUrl:    applicationParam.AppSpec.CallbackUrl,
	}

	logData.
		AddData("throttlingTier", appCreateReq.ThrottlingTier).
		AddData("description", appCreateReq.Description).
		AddData("callbackUrl", appCreateReq.CallbackUrl).
		AddData(LogKeyAPPName, appCreateReq.Name)
	utils.LogDebug("creating a new application...", logData)
	appID, err := asb.APIMManager.CreateApplication(appCreateReq)
	if err != nil {
		e, ok := err.(*client.InvokeError)
		if ok {
			logData.AddData("response code", strconv.Itoa(e.StatusCode))
		}
		utils.LogError("unable to create application", err, logData)
		return domain.ProvisionedServiceSpec{}, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionUnableToStoreInstance)
	}
	i := &dbutil.Instance{
		ServiceID:        serviceDetails.ServiceID,
		PlanID:           serviceDetails.PlanID,
		Id:               instanceID,
		APIMResourceID:   appID,
		APIMResourceName: applicationParam.AppSpec.Name,
	}
	logData.AddData(LogKeyAPPID, appID)
	// Store instance in the database
	err = dbutil.StoreInstance(i)
	if err != nil {
		utils.LogError("unable to store instance", err, logData)
		errDel := asb.APIMManager.DeleteApplication(appID)
		if errDel != nil {
			utils.LogError("failed to cleanup, unable to delete the application", errDel, logData)
		}
		return domain.ProvisionedServiceSpec{}, failureResponse500("unable to store instance in DB", "storing instance in DB")
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func (asb *APIMServiceBroker) createSubscriptionService(instanceID string, serviceDetails domain.ProvisionDetails) (spec domain.ProvisionedServiceSpec, err error) {
	var logData = &utils.LogData{
		Data: lager.Data{
			LogKeyServiceID:  serviceDetails.ServiceID,
			LogKeyPlanID:     serviceDetails.PlanID,
			LogKeyInstanceID: instanceID,
		},
	}
	// Verifying whether the instance is already exists
	exists, err := isInstanceExists(instanceID)
	// Handling DB connection error
	if err != nil {
		utils.LogError("unable to get instance from DB", err, logData)
		return domain.ProvisionedServiceSpec{}, failureResponse500("unable to query database", "getting the instance information from DB")
	}
	if exists {
		utils.LogError(CreateSubscriptionContext, apiresponses.ErrInstanceAlreadyExists, logData)
		return domain.ProvisionedServiceSpec{}, apiresponses.ErrInstanceAlreadyExists
	}
	subsInfo, err := toSubscriptionParam(serviceDetails.RawParameters)
	if err != nil {
		utils.LogError("unable to parse Subscription parameters", err, logData)
		return spec, invalidParamFailureResponse("parsing Subscription parameters")
	}
	if !utils.IsValidParams(subsInfo.SubsSpec.APIName, subsInfo.SubsSpec.AppName) {
		return domain.ProvisionedServiceSpec{}, invalidParamFailureResponse("parsing Subscription parameters")
	}
	logData.AddData("API name", subsInfo.SubsSpec.APIName).AddData("Application Name", subsInfo.SubsSpec.AppName)
	apiID, err := asb.APIMManager.SearchAPI(subsInfo.SubsSpec.APIName);
	if err != nil {
		utils.LogError("unable to search API", err, logData)
		return spec, failureResponse500(fmt.Sprintf("couldn't find the API: %s", subsInfo.SubsSpec.APIName), "searching API")
	}
	logData.AddData(LogKeyAPIID, apiID)
	appID, err := asb.APIMManager.SearchApplication(subsInfo.SubsSpec.AppName);
	if err != nil {
		utils.LogError("unable to search Application", err, logData)
		return spec, failureResponse500(fmt.Sprintf("couldn't find the Application: %s", subsInfo.SubsSpec.AppName), "searching application")
	}
	logData.AddData(LogKeyAPPID, appID)
	subsID, err := asb.APIMManager.Subscribe(appID, apiID, subsInfo.SubsSpec.SubscriptionTier)
	if err != nil {
		utils.LogError("unable to create the subscription", err, logData)
		return spec, failureResponse500("unable to create the subscription", "create subscription")
	}
	i := &dbutil.Instance{
		ServiceID:        serviceDetails.ServiceID,
		PlanID:           serviceDetails.PlanID,
		Id:               instanceID,
		APIMResourceID:   subsID,
		APIMResourceName: "",
	}
	logData.AddData(LogKeySubsID, subsID)
	// Store instance in the database
	err = dbutil.StoreInstance(i)
	if err != nil {
		utils.LogError("unable to store instance", err, logData)
		errDel := asb.APIMManager.UnSubscribe(subsID)
		if errDel != nil {
			utils.LogError("failed to cleanup, unable to delete the subscription", errDel, logData)
		}
		return spec, failureResponse500(ErrMSGUnableToStoreInstance, ErrActionUnableToStoreInstance)
	}
	return domain.ProvisionedServiceSpec{}, nil
}

func failureResponse500(m, a string) *apiresponses.FailureResponse {
	return apiresponses.NewFailureResponse(errors.New(m), http.StatusInternalServerError, a)
}

func invalidParamFailureResponse(a string) *apiresponses.FailureResponse {
	return apiresponses.NewFailureResponse(errors.New("invalid parameters"),
		http.StatusBadRequest, a)
}

func isAPIPlan(serviceID, planID string) bool {
	return (serviceID == constants.ServiceId) && (planID == constants.APIPlanID)
}

func isApplicationPlan(serviceID, planID string) bool {
	return (serviceID == constants.ServiceId) && (planID == constants.ApplicationPlanID)
}

func isSubscriptionPlan(serviceID, planID string) bool {
	return (serviceID == constants.ServiceId) && (planID == constants.SubscriptionPlanID)
}

// toApplicationParam parses application parameters
func toApplicationParam(params json.RawMessage) (ApplicationParam, error) {
	var a ApplicationParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, errors.Wrap(err, "unable to parse application parameters")
	}
	return a, nil
}

// toSubscriptionParam parses subscription parameters
func toSubscriptionParam(params json.RawMessage) (SubscriptionParam, error) {
	var s SubscriptionParam
	err := json.Unmarshal(params, &s)
	if err != nil {
		return s, errors.Wrap(err, "unable to parse subscription parameters")
	}
	return s, nil
}

// toAPIParam parses API spec parameter
func toAPIParam(params json.RawMessage) (APIParam, error) {
	var a APIParam
	err := json.Unmarshal(params, &a)
	if err != nil {
		return a, errors.Wrap(err, "unable to parse API parameters")
	}
	return a, nil
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
