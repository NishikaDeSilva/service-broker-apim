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

const (
	TableInstance = "instances"
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

// Create table for the Instance type
func (i *Instance) CreateTable() {
	if err := db.CreateTable(i).Error; err != nil {
		if mysqlErr, ok:= err.(*mysql.MySQLError); ok {
			if mysqlErr.Number != ErrTableExists {
				utils.HandleErrorWithLoggerAndExit(fmt.Sprintf("couldn't create the table :%s", i.Table()),err)
			}
		}
	}
}

// Table returns the Table name of the Instance
func (i *Instance) Table() string {
	return TableInstance
}

func (i *Instance) Retrieve() (bool, error) {
	result := db.Where(i).Find(i)
	if result.RecordNotFound() {
		return false, nil
	}
	return true, result.Error
}
