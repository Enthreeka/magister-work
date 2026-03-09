package service

import (
	"context"
	"fmt"

	"github.com/example/app/testdata/output/user_create/domain"
)

// Create implements the UserCreate business logic.
// This file is yours — codegen will never overwrite it.
func (s *UserCreateService) Create(ctx context.Context, req *domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrUserCreateValidation)
	}

	resp, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrUserCreateInternal, err)
	}

	return resp, nil

}
