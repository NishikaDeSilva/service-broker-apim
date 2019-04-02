/*
 *  Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 */

// dbutil package handles the DB connections and ORM
package dbutil

import (
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"time"
)
import "github.com/jinzhu/gorm"

const MySQL = "mysql"

type Instance struct {
}
type DBConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	MaxConnLifeTime time.Duration
	LogMode         bool
}

var db *gorm.DB

func InitDB(conf *DBConfig) (*gorm.DB, error) {
	url := conf.Username + ":" + conf.Password + "@tcp(" + conf.Host + ":" + strconv.Itoa(conf.Port) + ")/" +
		conf.Database + "?charset=utf8&parseTime=True&loc=Local"
	var err error
	db, err = gorm.Open(MySQL, url)
	if err != nil {
		return nil, err
	}
	db.DB().SetConnMaxLifetime(conf.MaxConnLifeTime)
	db.AutoMigrate(&Instance{})
	db.LogMode(conf.LogMode)
	return db, nil
}
