package service

import (
	"context"
)

// Store is an interface for handling the storing of permissions.
type Store interface {
	Create(ctx context.Context, permit Permit) (Permit, error)
	Get(ctx context.Context, filter interface{}) (Permit, error)
	GetAll(ctx context.Context, filter interface{}) ([]Permit, error)
	Delete(ctx context.Context, filter interface{}) (Permit, error)
	HealthCheck(ctx context.Context) (bool, error)
}
