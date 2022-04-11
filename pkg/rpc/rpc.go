package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/robotlovesyou/fitest/pb"
	"github.com/robotlovesyou/fitest/pkg/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UsersService defines the interface for the service RPCServer delegates its implementation logic to
type UsersService interface {
	CreateUser(context.Context, users.NewUser) (users.User, error)
}

// RPCServer is an impementation of generated.UsersService.
// It delegates all call handling logic to its UsersService, and is only responsible for converting
// back and forth between the types used by generated.UsersService and UsersService
type RPCServer struct {
	pb.UnimplementedUsersServer
	service UsersService
}

func New(service UsersService) *RPCServer {
	return &RPCServer{service: service}
}

func (svr *RPCServer) CreateUser(ctx context.Context, newUser *pb.NewUser) (*pb.User, error) {
	user, err := svr.service.CreateUser(ctx, users.NewUser{
		FirstName:       newUser.FirstName,
		LastName:        newUser.LastName,
		Nickname:        newUser.Nickname,
		Password:        newUser.Password,
		ConfirmPassword: newUser.ConfirmPassword,
		Email:           newUser.Email,
		Country:         newUser.Country,
	})
	if err != nil {
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, users.ErrAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, users.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "Internal Server Error")
		}
	}

	return &pb.User{
		Id:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}, nil
}
