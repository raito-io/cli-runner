package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/raito-io/raito-cli-container/constants"
)

type HealthChecker struct {
	livenessFilePath  string
	readinessFilePath string

	livenessFile  *os.File
	readinessFile *os.File
}

func NewHealthChecker() *HealthChecker {
	livenessFilePath := GetEnvString(constants.ENV_LIVENESS_FILE, "")

	return &HealthChecker{
		livenessFilePath: livenessFilePath,
	}
}

func (s *HealthChecker) MarkLiveness() error {
	if s.livenessFile != nil || s.livenessFilePath == "" {
		return nil
	}

	livenessFile, _, err := createOutputFile(s.livenessFilePath)
	if err != nil {
		return err
	}

	logrus.Infof("[Healthchecker] Created liveness file: %s", s.livenessFilePath)

	s.livenessFile = livenessFile

	return nil
}

func (s *HealthChecker) RemoveLivenessMark() error {
	if s.livenessFile != nil {
		err := s.livenessFile.Close()

		if err != nil {
			logrus.Errorf("failed to close file: %v", err)
			return err
		}

		err = os.Remove(s.livenessFilePath)
		if err != nil {
			logrus.Errorf("failed to remove file: %v", err)
			return err
		}

		s.livenessFile = nil
	}
	return nil
}

func (s *HealthChecker) Cleanup() {
	s.RemoveLivenessMark()
}
