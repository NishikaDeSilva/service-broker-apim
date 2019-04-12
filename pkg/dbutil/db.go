/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// dbutil package handles the DB connections and ORM
package dbutil

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/wso2/service-broker-apim/pkg/config"
	"github.com/wso2/service-broker-apim/pkg/utils"
	"strconv"
	"sync"
)
import "github.com/jinzhu/gorm"

const MySQL = "mysql"

// Instance represents the Instance model in the Database
type Instance struct {
	InstanceID string `gorm:"primary_key;type:varchar(100)"`
	ServiceID  string `gorm:"type:varchar(100)"`
	PlanID     string `gorm:"type:varchar(100)"`
	ApimID     string `gorm:"type:varchar(100)"`
}

// Bind represents the Bind model in the Database
type Bind struct {
	InstanceID   string `gorm:"type:varchar(100)"`
	BindID       string `gorm:"primary_key;type:varchar(100)"`
	AccessToken  string `gorm:"type:varchar(100)"`
	RefreshToken string `gorm:"type:varchar(100)"`
	AppName      string `gorm:"type:varchar(100)"`
	AppId        string `gorm:"type:varchar(100)"`
}

const (
	TableInstance  = "instances"
	TableBind      = "binds"
	ErrTableExists = 1050
)

var (
	onceInit sync.Once
	db       *gorm.DB
)

// Initialize database ORM
func InitDB(conf *config.DBConfig) {
	onceInit.Do(func() {
		url := conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
			conf.Database + "?charset=utf8&parseTime=True&loc=Local"
		var err error
		db, err = gorm.Open(MySQL, url)
		if err != nil {
			utils.HandleErrorWithLoggerAndExit("cannot initiate ORM", err)
		}
		//db.AutoMigrate(&Instance{})
		db.LogMode(conf.LogMode)
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

// Retrieve the given Instance from the Database
func Retrieve(model interface{}) (bool, error) {
	result := db.Where(model).Find(model)
	if result.RecordNotFound() {
		return false, nil
	}
	return true, result.Error
}

// Store save the given Instance in the Database
func Store(model interface{}, table string) error {
	return db.Table(table).Save(model).Error
}
