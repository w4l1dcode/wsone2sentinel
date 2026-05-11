package main

import (
	"context"
	"flag"
	"github.com/sirupsen/logrus"
	config2 "github.com/w4l1dcode/wsone2sentinel/config"
	msSentinel "github.com/w4l1dcode/wsone2sentinel/pkg/sentinel"
	ws1 "github.com/w4l1dcode/wsone2sentinel/pkg/wsone"
)

func main() {
	ctx := context.Background()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	confFile := flag.String("config", "config.yml", "The YAML configuration file.")
	flag.Parse()

	config, err := config2.LoadConfig(logrus.StandardLogger(), *confFile)
	if err != nil {
		logrus.Fatalf("could not load configuration: %s", err)
	}

	if err := config.Validate(); err != nil {
		logrus.WithError(err).Fatal("invalid configuration")
	}

	logrusLevel, err := logrus.ParseLevel(config.Log.Level)
	if err != nil {
		logger.WithError(err).Error("invalid log level provided")
		logrusLevel = logrus.InfoLevel
	}
	logger.SetLevel(logrusLevel)

	//

	ws1DeviceLogs, err := ws1.GetLogs(config, ctx)
	if err != nil {
		logrus.WithError(err).Fatal("could not get WS1 messages")
	}

	sentinel, err := msSentinel.New(logger, msSentinel.Credentials{
		TenantID:       config.Microsoft.TenantID,
		ClientID:       config.Microsoft.AppID,
		ClientSecret:   config.Microsoft.SecretKey,
		SubscriptionID: config.Microsoft.SubscriptionID,
		ResourceGroup:  config.Microsoft.ResourceGroup,
		WorkspaceName:  config.Microsoft.WorkspaceName,
	})
	if err != nil {
		logger.WithError(err).Fatal("could not create MS Sentinel client")
	}

	logger.WithField("total", len(ws1DeviceLogs)).Info("collected all WS1 logs")

	logger.WithField("total", len(ws1DeviceLogs)).Info("shipping WS1 device details to Sentinel")

	if err := sentinel.SendLogs(ctx, logger,
		config.Microsoft.DataCollection.Endpoint,
		config.Microsoft.DataCollection.RuleID,
		config.Microsoft.DataCollection.StreamName,
		ws1DeviceLogs); err != nil {
		logger.WithError(err).Fatal("could not ship logs to sentinel")
	}

	logger.WithField("total", len(ws1DeviceLogs)).Info("successfully sent logs to sentinel")

}
