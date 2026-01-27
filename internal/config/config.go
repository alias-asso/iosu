package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const PlateformName = "iosu"

type SqliteDb struct {
	DbPath string `toml:"db_path"`
}

type PostgresDb struct {
	DbUrl    string
	Username string
	Password string
}

type MysqlDb struct {
	DbUrl    string
	Username string
	Password string
}

type Config struct {
	ServerPort           string     `toml:"server_port"`
	JwtKey               string     `toml:"jwt_key"`
	DefaultAdminPassword string     `toml:"default_admin_password"`
	DbType               string     `toml:"db_type"`
	Sqlite               SqliteDb   `toml:"sqlite"`
	Mysql                MysqlDb    `toml:"mysql"`
	Postgres             PostgresDb `toml:"postgres"`
}

func ParseConfig(path string) (Config, error) {
	var config Config
	tomlData, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	_, err = toml.Decode(string(tomlData), &config)
	if err != nil {
		return Config{}, err
	}

	switch config.DbType {
	case "sqlite":
		if config.Sqlite == (SqliteDb{}) {
			return Config{}, errors.New("Empty sqlite config.")
		}
	case "mysql":
		if config.Mysql == (MysqlDb{}) {
			return Config{}, errors.New("Empty mysql config.")
		}
	case "postgres":
		if config.Postgres == (PostgresDb{}) {
			return Config{}, errors.New("Empty postgres config.")
		}
	default:
		return Config{}, errors.New("Unsupported database.")
	}

	return config, nil
}

func DefaultConfig() Config {
	return Config{
		Sqlite: SqliteDb{
			DbPath: fmt.Sprintf("/var/%s/db.sqlite", PlateformName),
		},
	}
}
