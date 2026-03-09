package service

import (
	"context"

	"github.com/example/app/testdata/output/user/domain"
)

// Create implements domain.UserServiceIface.
// TODO: replace with your business logic.
func (s *UserService) Create(ctx context.Context, req *domain.UserRequest) (*domain.UserResponse, error) {
	panic("not implemented: Create")
}
