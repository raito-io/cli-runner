package main

import (
	"context"

	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

func main() {
	ctx := context.Background()

	err := viper.BindEnv(constants.ENV_UPDATE_CRON)
	if err != nil {
		panic(err)
	}

	githubRepo := github.NewGithubRepo()

	service, cleanup, err := NewService(githubRepo)
	if err != nil {
		panic(err)
	}

	defer cleanup()

	err = service.Run(ctx)
	if err != nil {
		panic(err)
	}
}
