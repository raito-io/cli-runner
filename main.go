package main

import (
	"context"

	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

func main() {
	ctx := context.Background()

	viper.BindEnv(constants.UPDATE_CRON)

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
