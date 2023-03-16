package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/raito-io/raito-cli-container/constants"
	"github.com/raito-io/raito-cli-container/github"
)

const workingdir = "./"

type Service struct {
	githubRepo *github.GithubRepo
	scheduler  *cron.Cron

	mutex             sync.Mutex
	cmd               *exec.Cmd
	version           *semver.Version
	executionLocation string

	waitGroup sync.WaitGroup
}

func NewService(githubRepo *github.GithubRepo) (*Service, error) {
	return &Service{
		githubRepo: githubRepo,
		scheduler:  cron.New(),
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

	s.scheduler.AddFunc(s.getCronSpec(), func() {
		err := s.cliVersionCheck(ctx)
		if err != nil {
			logrus.Error(err)
		}

		s.logNextUpdateCheck()
	})

	s.scheduler.Start()

	s.logNextUpdateCheck()

	s.waitGroup.Wait()

	return nil
}

func (s *Service) getCronSpec() string {
	if viper.IsSet(constants.UPDATE_CRON) {
		cron := viper.GetString(constants.UPDATE_CRON)
		return cron
	} else {
		logrus.Info("Updating cron every day at 2:00")
		return "0 2 * * *"
	}
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

			s.cmd = exec.CommandContext(ctx, execLocation, os.Args[1:]...)
			s.cmd.Stdout = os.Stdout
			s.cmd.Stderr = os.Stderr
			s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			logrus.Infof("Executing CLI version %s with command: %s", s.version.String(), s.cmd.String())

			s.mutex.Unlock()

			s.cmd.Run()

			logrus.Info("Finished executing CLI")
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
		logrus.Infof("Found new CLI version %s", latestVersion.String())

		version, location, err := s.githubRepo.InstallLatestRelease(ctx, workingdir)
		if err != nil {
			logrus.Errorf("Failed to install latest release: %v", err)

			return err
		}

		previousLocation := s.executionLocation
		s.version = version
		s.executionLocation = location

		logrus.Debug("Stop previous runner")

		err = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
		if err != nil {
			logrus.Errorf("%v", err)
			return err
		}

		logrus.Debug("process is stopped")

		logrus.Debug("Remove previous runner")
		if previousLocation != location {
			err = os.Remove(previousLocation)
			if err != nil {
				return err
			}
		}

	} else {
		logrus.Info("CLI version is up to date")
	}

	return nil
}

func (s *Service) logNextUpdateCheck() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.scheduler.Entries()) > 0 {
		t := s.scheduler.Entries()[0].Next
		logrus.Infof("Next update check at %s", t.Format(time.RFC822))
	} else {
		logrus.Info("No jobs scheduled")
	}
}