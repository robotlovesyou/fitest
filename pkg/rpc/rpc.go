package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/robotlovesyou/fitest/pb"
	"github.com/robotlovesyou/fitest/pkg/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// UsersService defines the interface for the service RPCServer delegates its implementation logic to
type UsersService interface {
	CreateUser(context.Context, users.NewUser) (users.User, error)
	UpdateUser(context.Context, users.UserUpdate) (users.User, error)
	DeleteUser(context.Context, users.UserRef) error
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

func pbUserFromUser(user *users.User) *pb.User {
	return &pb.User{
		Id:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Country:   user.Country,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
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

	return pbUserFromUser(&user), nil
}

func (svr *RPCServer) UpdateUser(ctx context.Context, userUpdate *pb.UserUpdate) (*pb.User, error) {
	user, err := svr.service.UpdateUser(ctx, users.UserUpdate{
		ID:              userUpdate.Id,
		FirstName:       userUpdate.FirstName,
		LastName:        userUpdate.LastName,
		Password:        userUpdate.Password,
		ConfirmPassword: userUpdate.ConfirmPassword,
		Country:         userUpdate.Country,
		Version:         userUpdate.Version,
	})
	if err != nil {
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, users.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, users.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, users.ErrInvalidVersion):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, "Internal Server Error")
		}
	}
	return pbUserFromUser(&user), nil
}

func (svr *RPCServer) DeleteUser(ctx context.Context, userRef *pb.UserRef) (*emptypb.Empty, error) {
	if err := svr.service.DeleteUser(ctx, users.UserRef{ID: userRef.Id}); err != nil {
		panic("error handling not implemented")
	}
	return &emptypb.Empty{}, nil
}
