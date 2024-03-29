package main

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

func start(ctx context.Context) error {
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

	return service.Run(ctx)
}

func main() {
	ctx := context.Background()

	err := start(ctx)
	if err != nil {
		logrus.Errorf("execution error: %s", err.Error())
		eError := &exec.ExitError{}

		if errors.As(err, &eError) {
			exitCode := eError.ExitCode()
			os.Exit(exitCode)
		} else {
			panic(err)
		}
	}
}
