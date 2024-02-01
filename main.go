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
	var cleanups []func()

	err := viper.BindEnv(constants.ENV_UPDATE_CRON)
	if err != nil {
		panic(err)
	}

	githubRepo := github.NewGithubRepo()
	healthChecker := NewHealthChecker()
	cleanups = append(cleanups, healthChecker.Cleanup)

	service, serviceCleanup, err := NewService(githubRepo, healthChecker)
	if err != nil {
		panic(err)
	}

	cleanups = append(cleanups, serviceCleanup)

	defer func() {
		for _, f := range cleanups {
			f()
		}
	}()

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
