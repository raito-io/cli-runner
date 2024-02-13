package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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

var signal = syscall.SIGUSR1

type Service struct {
	githubRepo    *github.GithubRepo
	healthChecker *HealthChecker
	scheduler     *cron.Cron

	mutex             sync.Mutex
	cmd               *exec.Cmd
	version           *semver.Version
	executionLocation string

	stdoutWriter io.Writer
	stderrWriter io.Writer

	waitGroup      sync.WaitGroup
	exitError      error
	exitErrorMutex sync.Mutex

	userSignal chan struct{}
	terminated chan struct{}
}

func NewService(githubRepo *github.GithubRepo, healthChecker *HealthChecker) (*Service, func(), error) {
	stdoutFileName := GetEnvString(constants.ENV_STDOUT_FILE, os.Stdout.Name())
	stderrFileName := GetEnvString(constants.ENV_STDERR_FILE, os.Stderr.Name())

	var cleanup []func() error

	stdoutFile, stdoutFileCleanup, err := createOutputFile(stdoutFileName)
	if err != nil {
		return nil, nil, err
	}

	cleanup = append(cleanup, stdoutFileCleanup)

	stderrFile, stderrFileCleanup, err := createOutputFile(stderrFileName)
	if err != nil {
		return nil, nil, err
	}

	cleanup = append(cleanup, stderrFileCleanup)

	return &Service{
			githubRepo:    githubRepo,
			healthChecker: healthChecker,
			scheduler:     cron.New(),

			userSignal: make(chan struct{}),
			terminated: make(chan struct{}, 1),

			stdoutWriter: stdoutFile,
			stderrWriter: stderrFile,
		}, func() {
			for _, f := range cleanup {
				err := f()

				if err != nil {
					logrus.Errorf("failed to close file: %v", err)
				}
			}
		}, nil
}

func (s *Service) Run(ctx context.Context) error {
	defer close(s.userSignal)
	defer close(s.terminated)

	// Start with downloading the latest release
	version, location, err := s.githubRepo.InstallLatestRelease(ctx, workingdir)
	if err != nil {
		return err
	}

	s.version = version
	s.executionLocation = location

	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.waitGroup.Add(1)

	go func() {
		defer s.waitGroup.Done()

		runErr := s.runRaitoCli(cancelCtx)
		if runErr != nil {
			logrus.Errorf("error while running Raito CLI: %v", runErr)

			s.exitErrorMutex.Lock()
			defer s.exitErrorMutex.Unlock()

			s.exitError = runErr

			s.healthChecker.RemoveLivenessMark()
		}
	}()

	_, err = s.scheduler.AddFunc(s.getCronSpec(), func() {
		err2 := s.cliVersionCheck(ctx)
		if err2 != nil {
			logrus.Error(err2)

			s.healthChecker.RemoveLivenessMark()
		}

		s.logNextUpdateCheck()
	})
	if err != nil {
		return fmt.Errorf("schedule update: %w", err)
	}

	s.scheduler.Start()

	defer func() { ctx = s.scheduler.Stop() }()

	s.logNextUpdateCheck()

	err = s.healthChecker.MarkLiveness()
	if err != nil {
		return err
	}

	s.waitGroup.Wait()

	s.exitErrorMutex.Lock()
	defer s.exitErrorMutex.Unlock()

	return s.exitError
}

func (s *Service) getCronSpec() string {
	if viper.IsSet(constants.ENV_UPDATE_CRON) {
		cron := viper.GetString(constants.ENV_UPDATE_CRON)
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
			s.cmd.Stdout = s.stdoutWriter
			s.cmd.Stderr = s.stderrWriter
			s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			logrus.Infof("Executing CLI version %s with command: %s", s.version.String(), s.cmd.String())

			s.mutex.Unlock()

			exitError := s.cmd.Run()
			if exitError != nil {
				logrus.Debugf("error while executing CLI: %v", exitError)

				eError := &exec.ExitError{}
				if errors.As(exitError, &eError) {
					ws := eError.ProcessState.Sys().(syscall.WaitStatus)
					if ws.ExitStatus() == int(signal) {
						logrus.Info("Restart RAITO CLI")

						s.userSignal <- struct{}{}

						continue
					}
				}

				logrus.Errorf("error while executing CLI: %s", exitError.Error())

				s.terminated <- struct{}{}

				return exitError
			}

			logrus.Info("Finished executing CLI")

			return nil
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

		err, done := s.stopCLI()
		if done {
			return err
		}

		logrus.Debug("Process is stopped")

		if previousLocation != location {
			logrus.Debug("Remove previous runner")
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

func (s *Service) stopCLI() (error, bool) {
	err := syscall.Kill(-s.cmd.Process.Pid, signal)
	if err != nil {
		logrus.Errorf("%v", err)
		return err, true
	}

	logrus.Debug("Wait for process to stop...")

	select {
	case <-s.terminated:
		return nil, true
	case <-s.userSignal:
	}

	return nil, false
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

func GetEnvString(key, defaultVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}

	return v
}

func createOutputFile(filename string) (*os.File, func() error, error) {
	if filename == os.Stdout.Name() {
		return os.Stdout, func() error { return nil }, nil
	} else if filename == os.Stderr.Name() {
		return os.Stderr, func() error { return nil }, nil
	}

	outputFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, err
	}

	return outputFile, func() error {
		return outputFile.Close()
	}, nil
}
