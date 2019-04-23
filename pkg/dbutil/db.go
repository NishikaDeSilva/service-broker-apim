/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// dbutil package handles the DB connections and ORM
package dbutil

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"log"
	"strconv"
	"sync"
)
import "github.com/jinzhu/gorm"

const MySQL = "mysql"

// Instance represents the Instance model in the Database
type Instance struct {
	InstanceID string `gorm:"primary_key;type:varchar(100)"`
	ServiceID  string `gorm:"type:varchar(100);not null"`
	PlanID     string `gorm:"type:varchar(100);not null"`
	ApiID      string `gorm:"type:varchar(100);not null;unique"`
	APIName    string `gorm:"type:varchar(100);not null;unique"`
}

// Application represents the Application model in the database
type Application struct {
	AppName        string `gorm:"primary_key;type:varchar(100)"`
	AppID          string `gorm:"type:varchar(100);not null;unique"`
	Token          string `gorm:"type:varchar(100)"`
	ConsumerKey    string `gorm:"type:varchar(100)"`
	ConsumerSecret string `gorm:"type:varchar(100)"`
	SubscriptionTier string `gorm:"type:varchar(100);not null"`
}

// Bind represents the Bind model in the Database
type Bind struct {
	BindID           string `gorm:"primary_key;type:varchar(100)"`
	SubscriptionID   string `gorm:"type:varchar(100);not null;unique"`
	InstanceID       string `gorm:"type:varchar(100);not null"`
	AppName          string `gorm:"type:varchar(100);not null"`
	ServiceID        string `gorm:"type:varchar(100);not null"`
	PlanID           string `gorm:"type:varchar(100);not null"`
}

const (
	TableInstance           = "instances"
	TableBind               = "binds"
	TableApplication        = "applications"
	ErrTableExists          = 1050
	ErrMSGAPPNameMissing    = "AppName is missing"
	ErrMSGInstanceIDMissing = "InstanceID is missing"
	ErrMSGBindIDMissing     = "BindID is missing"
)

var (
	onceInit sync.Once
	db       *gorm.DB
)

// Initialize database ORM
func InitDB(conf *config.DBConfig) {
	onceInit.Do(func() {
		url := conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
			conf.Database + "?charset=utf8"
		var err error
		db, err = gorm.Open(MySQL, url)
		if err != nil {
			utils.HandleErrorWithLoggerAndExit("cannot initiate ORM", err)
		}
		db.LogMode(conf.LogMode)
		ioWriter := utils.IoWriterLog()
		if ioWriter == nil {
			utils.HandleErrorWithLoggerAndExit("cannot setup logger for DB", errors.New("IoWriter for logging is not initialized"))
		}
		db.SetLogger(gorm.Logger{LogWriter: log.New(ioWriter, "database", 0)})
	})
}

// Creates a table for the given model only if table already not exists
func CreateTable(model interface{}, table string) {
	if err := db.CreateTable(model).Error; err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			// Not throw error if the table already exists
			if mysqlErr.Number != ErrTableExists {
				utils.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", table), err)
			}
		}
	}
}

// store save the given Instance in the Database
func store(model interface{}, table string) error {
	return db.Table(table).Create(model).Error
}

// update updates the given Instance in the Database
func update(model interface{}, table string) error {
	return db.Table(table).Save(model).Error
}

// deleteEntry deletes the given Instance in the Database
func deleteEntry(model interface{}, table string) error {
	return db.Table(table).Delete(model).Error
}

// retrieve function returns the given Instance from the Database ix exists
func retrieve(model interface{}, table string) (bool, error) {
	result := db.Table(table).Where(model).Find(model)
	if result.RecordNotFound() {
		return false, nil
	}
	return true, result.Error
}

// addForeignKeys configures foreign keys
func addForeignKeys() {
	err := db.Model(&Bind{}).
		AddForeignKey("app_name", TableApplication+"(app_name)", "CASCADE",
			"CASCADE").
		AddForeignKey("instance_id", TableInstance+"(instance_id)", "CASCADE",
			"CASCADE").Error
	if err != nil {
		utils.HandleErrorWithLoggerAndExit("unable to add foreign keys", err)
	}
}

// CreateInstanceTable creates the Instance tables
func CreateTables() {
	CreateTable(&Instance{}, TableInstance)
	CreateTable(&Application{}, TableApplication)
	CreateTable(&Bind{}, TableBind)
	addForeignKeys()
}

// RetrieveInstance function returns the given Instance from the Database
func RetrieveInstance(i *Instance) (bool, error) {
	if i.InstanceID == "" {
		return false, errors.New(ErrMSGInstanceIDMissing)
	}
	return retrieve(i, TableInstance)
}

// StoreInstance saves the Instance in the database
func StoreInstance(i *Instance) error {
	if i.InstanceID == "" {
		return errors.New(ErrMSGInstanceIDMissing)
	}
	return store(i, TableInstance)
}

// DeleteInstance deletes the Instance in the database
func DeleteInstance(i *Instance) error {
	if i.InstanceID == "" {
		return errors.New(ErrMSGInstanceIDMissing)
	}
	return deleteEntry(i, TableInstance)
}

// RetrieveBind function returns the given Bind from the Database
func RetrieveBind(b *Bind) (bool, error) {
	if b.BindID == "" {
		return false, errors.New(ErrMSGBindIDMissing)
	}
	return retrieve(b, TableBind)
}

// StoreBind saves the Bind in the database
func StoreBind(b *Bind) error {
	if b.BindID == "" {
		return errors.New(ErrMSGBindIDMissing)
	}
	return store(b, TableBind)
}

// DeleteBind deletes the Bind in the database
func DeleteBind(b *Bind) error {
	if b.BindID == "" {
		return errors.New(ErrMSGBindIDMissing)
	}
	return deleteEntry(b, TableBind)
}

// RetrieveApp function returns the given Application from the Database
func RetrieveApp(a *Application) (bool, error) {
	if a.AppName == "" {
		return false, errors.New(ErrMSGAPPNameMissing)
	}
	return retrieve(a, TableApplication)
}

// StoreApp saves the Application in the database
func StoreApp(b *Application) error {
	if b.AppName == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return store(b, TableApplication)
}

// UpdateApp updates the application entry
func UpdateApp(b *Application) error {
	if b.AppName == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return update(b, TableApplication)
}

// DeleteApp deletes the application entry
func DeleteApp(b *Application) error {
	if b.AppName == "" {
		return errors.New(ErrMSGAPPNameMissing)
	}
	return deleteEntry(b, TableApplication)
}
