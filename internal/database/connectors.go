package database

import (
	"errors"

	"github.com/alias-asso/iosu/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectSqlite(path string, config *config.Config) (error, *gorm.DB) {
	var gormConfig *gorm.Config = &gorm.Config{}
	if !config.DevMode {
		gormConfig.Logger.LogMode(logger.Silent)
	}
	db, err := gorm.Open(sqlite.Open(path), gormConfig)
	if err != nil {
		return err, &gorm.DB{}
	}
	return nil, db
}

func ConnectDb(config *config.Config) (error, *gorm.DB) {
	var db *gorm.DB
	switch config.DbType {
	case "sqlite":
		err, conn := ConnectSqlite(config.Sqlite.DbPath, config)
		if err != nil {
			return err, &gorm.DB{}
		}
		db = conn
	case "postgres":
		// TODO
		return errors.ErrUnsupported, &gorm.DB{}

	case "mysql":
		// TODO
		return errors.ErrUnsupported, &gorm.DB{}

	default:
		// TODO
		return errors.ErrUnsupported, &gorm.DB{}
	}
	return nil, db
}
