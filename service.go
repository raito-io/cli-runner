package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

const workingdir = "./"

type Service struct {
	githubRepo *github.GithubRepo
	scheduler  *gocron.Scheduler

	mutex             sync.Mutex
	cmd               *exec.Cmd
	version           *semver.Version
	executionLocation string

	waitGroup sync.WaitGroup
}

func NewService(githubRepo *github.GithubRepo) (*Service, error) {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		return nil, err
	}

	s := gocron.NewScheduler(location)

	if viper.IsSet(constants.UPDATE_CRON) {
		s.Cron(viper.GetString(constants.UPDATE_CRON))
	} else {
		s.Every(1).Day().At("02:00")
	}

	return &Service{
		githubRepo: githubRepo,
		scheduler:  s,
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	// Start with downloading the latest release
	version, location, err := s.githubRepo.InstallLatestRelease(ctx, workingdir)
	if err != nil {
		return err
	}

	s.version = version
	s.executionLocation = location

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		s.waitGroup.Add(1)
		defer s.waitGroup.Done()

		runErr := s.runRaitoCli(cancelCtx)
		if runErr != nil {
			logrus.Errorf("error while running Raito CLI: %v", runErr)
		}
	}()

	s.scheduler.Do(func() {
		err := s.cliVersionCheck(ctx)
		if err != nil {
			logrus.Error(err)
		}
	})

	s.scheduler.StartAsync()

	<-ctx.Done()

	return nil
}

func (s *Service) runRaitoCli(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			var execLocation string
			s.mutex.Lock()
			if s.executionLocation != "" {
				execLocation = s.executionLocation
			} else {
				return errors.New("no execution location")
			}

			s.cmd = exec.Command(execLocation, os.Args[1:]...)
			s.cmd.Stdout = os.Stdout
			s.cmd.Stderr = os.Stderr
			s.cmd.Stdin = &NullReader{}

			logrus.Infof("Executing CLI version %s with command: %s", s.version.String(), s.cmd.String())

			s.mutex.Unlock()

			err := s.cmd.Start()
			if err != nil {
				return err
			}

			err = s.cmd.Wait()
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) cliVersionCheck(ctx context.Context) error {
	logrus.Info("Checking for RAITO CLI update")

	s.waitGroup.Add(1)
	defer s.waitGroup.Done()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	latestVersion, err := s.githubRepo.GetLatestReleasedVersion(ctx)
	if err != nil {
		logrus.Errorf("Failed to get latest released version: %v", err)

		return err
	}

	if latestVersion.GreaterThan(s.version) {
		logrus.Info("Found new CLI version %s", latestVersion.String())

		version, location, err := s.githubRepo.InstallLatestRelease(ctx, workingdir)
		if err != nil {
			return err
		}

		previousLocation := s.executionLocation
		s.version = version
		s.executionLocation = location

		err = s.cmd.Process.Kill()
		if err != nil {
			return err
		}

		err = os.Remove(previousLocation)
		if err != nil {
			return err
		}

	} else {
		logrus.Info("CLI version is up to date")
	}

	return nil
}

type NullReader struct {
}

func (r *NullReader) Read(p []byte) (n int, err error) {
	return 0, nil
}
