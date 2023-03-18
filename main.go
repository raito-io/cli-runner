package main

import (
	"context"

	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

func main() {
	ctx := context.Background()

	err := viper.BindEnv(constants.UPDATE_CRON)
	if err != nil {
		panic(err)
	}

	githubRepo := github.NewGithubRepo()

	service, err := NewService(githubRepo)
	if err != nil {
		panic(err)
	}

	err = service.Run(ctx)
	if err != nil {
		panic(err)
	}
}
