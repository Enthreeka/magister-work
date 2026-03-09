package service

import (
	"context"
	"fmt"

	"github.com/example/app/testdata/output/user/domain"
)

// Create implements the User business logic.
// This file is yours — codegen will never overwrite it.
func (s *UserService) Create(ctx context.Context, req *domain.UserRequest) (*domain.UserResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrUserValidation)
	}
	if req.Email == "" {
		return nil, fmt.Errorf("%w: email is required", domain.ErrUserValidation)
	}

	resp, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUserInternal, err)
	}

	return resp, nil

}
