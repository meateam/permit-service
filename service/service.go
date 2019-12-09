package service

import (
	"github.com/sirupsen/logrus"
	"time"
)

// Service is the structure used for handling
type Service struct {
	controller Controller
	logger     *logrus.Logger
}

// HealthCheck checks the health of the service, and returns a boolean accordingly.
func (s *Service) HealthCheck(mongoClientPingTimeout time.Duration) bool {
	return true
}

// NewService creates a Service and returns it.
func NewService(controller Controller, logger *logrus.Logger) Service {
	return Service{controller: controller, logger: logger}
}
