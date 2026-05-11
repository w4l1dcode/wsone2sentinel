package config

import (
	"fmt"
	validator "github.com/asaskevich/govalidator"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	defaultLogLevel = "DEBUG"
	appEnvPrefix    = "CSS"
)

type Config struct {
	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL" valid:"optional"`
	} `json:"log"`

	Microsoft struct {
		AppID          string `yaml:"app_id" env:"MS_APP_ID" valid:"required"`
		SecretKey      string `yaml:"secret_key" env:"MS_SECRET_KEY" valid:"required"`
		TenantID       string `yaml:"tenant_id" env:"MS_TENANT_ID" valid:"required"`
		SubscriptionID string `yaml:"subscription_id" env:"MS_SUB_ID" valid:"required"`

		DataCollection struct {
			Endpoint   string `yaml:"endpoint" env:"MS_DCR_ENDPOINT" valid:"minstringlength(3)"`
			RuleID     string `yaml:"rule_id" env:"MS_DCR_RULE" valid:"minstringlength(3)"`
			StreamName string `yaml:"stream_name" env:"MS_DCR_STREAM" valid:"minstringlength(3)"`
		} `yaml:"dcr"`

		ResourceGroup string `yaml:"resource_group" env:"MS_RSG_ID" valid:"required"`
		WorkspaceName string `yaml:"workspace_name" env:"MS_WS_NAME" valid:"required"`
	} `yaml:"microsoft"`

	WS1 struct {
		Endpoint     string `yaml:"api_url" env:"WS1_API_URL"`
		AuthLocation string `yaml:"auth_location" env:"WS1_AUTH_LOCATION"`
		ClientID     string `yaml:"client_id" env:"WS1_CLIENT_ID"`
		ClientSecret string `yaml:"client_secret" env:"WS1_CLIENT_SECRET"`

		SkipFilters []struct {
			Policy string `yaml:"policy"`
			User   string `yaml:"user"`
		} `yaml:"skip"`
	} `yaml:"ws1"`
}

func LoadConfig(logger *logrus.Logger, path string) (*Config, error) {
	var config Config

	if path != "" {
		configBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not load configuration file")
		}

		if err := yaml.Unmarshal(configBytes, &config); err != nil {
			return nil, errors.Wrap(err, "could not parse configuration file")
		}

		logger.Info("loaded configuration from " + path)
	}

	if err := envconfig.Process(appEnvPrefix, &config); err != nil {
		return nil, errors.Wrap(err, "could not load environment variables")
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if c.Log.Level == "" {
		c.Log.Level = defaultLogLevel
	}

	if valid, err := validator.ValidateStruct(c); !valid || err != nil {
		return fmt.Errorf("invalid configuration: %v", err)
	}

	return nil
}

func (c *Config) Save(logger *logrus.Logger, path string) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "could not serialize config")
	}

	if err := os.WriteFile(path, b, 0600); err != nil {
		return errors.Wrap(err, "could not write config")
	}

	return nil
}
