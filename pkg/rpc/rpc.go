package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/robotlovesyou/fitest/pkg/users"
	"github.com/robotlovesyou/fitest/userspb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// UsersService defines the interface for the service RPCServer delegates its implementation logic to
type UsersService interface {
	CreateUser(context.Context, users.NewUser) (users.User, error)
	UpdateUser(context.Context, users.Update) (users.User, error)
	DeleteUser(context.Context, users.Ref) error
	FindUsers(context.Context, users.Query) (users.Page, error)
}

// RPCServer is an impementation of generated.UsersService.
// It delegates all call handling logic to its UsersService, and is only responsible for converting
// back and forth between the types used by generated.UsersService and UsersService
type RPCServer struct {
	userspb.UnimplementedUsersServer
	service UsersService
}

func New(service UsersService) *RPCServer {
	return &RPCServer{service: service}
}

func pbUserFromUser(user *users.User) *userspb.User {
	return &userspb.User{
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

func pbPageFromPage(page *users.Page) *userspb.Page {
	items := make([]*userspb.User, 0, len(page.Items))
	for _, itm := range page.Items {
		items = append(items, pbUserFromUser(&itm))
	}
	return &userspb.Page{
		Page:  page.Page,
		Total: page.Total,
		Items: items,
	}
}

func (svr *RPCServer) CreateUser(ctx context.Context, newUser *userspb.NewUser) (*userspb.User, error) {
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

func (svr *RPCServer) UpdateUser(ctx context.Context, userUpdate *userspb.Update) (*userspb.User, error) {
	user, err := svr.service.UpdateUser(ctx, users.Update{
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

func (svr *RPCServer) DeleteUser(ctx context.Context, userRef *userspb.Ref) (*emptypb.Empty, error) {
	if err := svr.service.DeleteUser(ctx, users.Ref{ID: userRef.Id}); err != nil {
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, users.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, users.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "Internal Server Error")
		}
	}
	return &emptypb.Empty{}, nil
}

func (svr *RPCServer) FindUsers(ctx context.Context, query *userspb.Query) (*userspb.Page, error) {
	page, err := svr.service.FindUsers(ctx, users.Query{
		CreatedAfter: query.CreatedAfter,
		Country:      query.Country,
		Length:       query.Length,
		Page:         query.Page,
	})
	if err != nil {
		panic("error handling not implemented")
	}
	return pbPageFromPage(&page), nil
}
