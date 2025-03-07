package main

import (
	"github.com/urfave/cli"
	"github.com/0glabs/0g-data-avail/cli/flags"
	"github.com/0glabs/0g-data-avail/common/aws"
	"github.com/0glabs/0g-data-avail/common/logging"
)

type Config struct {
	AwsClientConfig aws.ClientConfig
	LoggerConfig    logging.Config
}

func NewConfig(ctx *cli.Context) *Config {
	return &Config{
		AwsClientConfig: aws.ReadClientConfig(ctx, flags.FlagPrefix),
		LoggerConfig:    logging.ReadCLIConfig(ctx, flags.FlagPrefix),
	}
}
