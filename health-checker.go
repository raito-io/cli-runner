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
	readinessFilePath := GetEnvString(constants.ENV_READINESS_FILE, "")

	return &HealthChecker{
		livenessFilePath:  livenessFilePath,
		readinessFilePath: readinessFilePath,
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

func (s *HealthChecker) MarkReadiness() error {
	if s.readinessFile != nil || s.readinessFilePath == "" {
		return nil
	}

	readinessFile, _, err := createOutputFile(s.readinessFilePath)
	if err != nil {
		return err
	}

	logrus.Infof("[Healthchecker] Created readiness file: %s", s.readinessFilePath)

	s.readinessFile = readinessFile

	return nil
}

func (s *HealthChecker) Cleanup() {
	if s.livenessFile != nil {
		err := s.livenessFile.Close()
		if err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}

		err = os.Remove(s.livenessFilePath)
		if err != nil {
			logrus.Errorf("failed to remove file: %v", err)
		}
	}

	if s.readinessFile != nil {
		err := s.readinessFile.Close()
		if err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}

		err = os.Remove(s.readinessFilePath)
		if err != nil {
			logrus.Errorf("failed to remove file: %v", err)
		}
	}
}
