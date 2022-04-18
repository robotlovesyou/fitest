// package rpc implements a userspb.UsersServer, depending on an implementation of UsersService
// to process requests
package rpc

import (
	"context"
	"errors"
	"time"

	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/robotlovesyou/fitest/pkg/telemetry"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/robotlovesyou/fitest/userspb"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// Error message sent for internal errors
	msgInternalServerError = "Internal Server Error"
)

// UsersService defines the interface for the service RPCServer delegates its implementation logic to
type UsersService interface {
	CreateUser(context.Context, *user.NewUser) (user.User, error)
	UpdateUser(context.Context, *user.Update) (user.User, error)
	DeleteUser(context.Context, *user.Ref) error
	FindUsers(context.Context, *user.Query) (user.Page, error)
}

// RPCServer is an impementation of userspb.UsersService.
// It delegates all call handling logic to its UsersService, and is only responsible for converting
// back and forth between the types used by generated.UsersService and UsersService
type RPCServer struct {
	userspb.UnimplementedUsersServer
	service UsersService
	// In a production setting I would declare this as an interface to allow for stub implementations for testing
	logger *log.Logger
}

// New creates a new RPCServer which will delegate processing to its UsersService dependency
func New(service UsersService, logger *log.Logger) *RPCServer {
	return &RPCServer{service: service, logger: logger}
}

// pbUserFromUser converts a user.User into a userspb.User
func pbUserFromUser(user *user.User) *userspb.User {
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

// pbPageFromPage converts a user.Page into a userspb.Page
func pbPageFromPage(page *user.Page) *userspb.Page {
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

// CreateUser implements the userspb.UsersServer.CreateUser function, allowing clients to create new users
func (svr *RPCServer) CreateUser(ctx context.Context, newUser *userspb.NewUser) (*userspb.User, error) {
	// placing the email in the logs like this could be a GDPR issue, depending on company policy
	ctx, span := otel.Tracer(telemetry.TraceName).Start(ctx, "CreateUser")
	defer span.End()
	svr.logger.Infof(ctx, "creating user %s", newUser.Email)

	usr, err := svr.service.CreateUser(ctx, &user.NewUser{
		FirstName:       newUser.FirstName,
		LastName:        newUser.LastName,
		Nickname:        newUser.Nickname,
		Password:        newUser.Password,
		ConfirmPassword: newUser.ConfirmPassword,
		Email:           newUser.Email,
		Country:         newUser.Country,
	})
	if err != nil {
		svr.logger.Errorf(ctx, err, "error creating user %s", newUser.Email)
		span.RecordError(err)
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, user.ErrAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, user.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, msgInternalServerError)
		}
	}

	return pbUserFromUser(&usr), nil
}

// UpdateUser implements the userspb.UsersServer.UpdateUser function, allowing clients to update existing users
func (svr *RPCServer) UpdateUser(ctx context.Context, userUpdate *userspb.Update) (*userspb.User, error) {
	ctx, span := otel.Tracer(telemetry.TraceName).Start(ctx, "UpdateUser")
	defer span.End()
	svr.logger.Infof(ctx, "updating user %s", userUpdate.Id)

	defer span.End()
	usr, err := svr.service.UpdateUser(ctx, &user.Update{
		ID:              userUpdate.Id,
		FirstName:       userUpdate.FirstName,
		LastName:        userUpdate.LastName,
		Password:        userUpdate.Password,
		ConfirmPassword: userUpdate.ConfirmPassword,
		Country:         userUpdate.Country,
		Version:         userUpdate.Version,
	})
	if err != nil {
		svr.logger.Errorf(ctx, err, "error updating user %s", userUpdate.Id)
		span.RecordError(err)
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, user.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, user.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, user.ErrInvalidVersion):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		default:
			return nil, status.Error(codes.Internal, msgInternalServerError)
		}
	}
	return pbUserFromUser(&usr), nil
}

// DeleteUser implements the userspb.UsersServer.DeleteUser function, allowing clients to delete users
func (svr *RPCServer) DeleteUser(ctx context.Context, userRef *userspb.Ref) (*emptypb.Empty, error) {
	ctx, span := otel.Tracer(telemetry.TraceName).Start(ctx, "DeleteUser")
	defer span.End()
	svr.logger.Infof(ctx, "deleting user %s", userRef.Id)

	if err := svr.service.DeleteUser(ctx, &user.Ref{ID: userRef.Id}); err != nil {
		svr.logger.Errorf(ctx, err, "error deleting user: %s", userRef.Id)
		span.RecordError(err)
		// For the sake of brevity, I am only going to use grpc error codes when the service fails.
		// In a real world implementation I would, where appropriate, include detail via status details.
		switch {
		case errors.Is(err, user.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, user.ErrInvalid):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, msgInternalServerError)
		}
	}
	return &emptypb.Empty{}, nil
}

// FindUsers implements the userspb.UsersServer.FindUsers function, allowing clients to find users and page through results
func (svr *RPCServer) FindUsers(ctx context.Context, query *userspb.Query) (*userspb.Page, error) {
	ctx, span := otel.Tracer(telemetry.TraceName).Start(ctx, "FindUsers")
	defer span.End()
	svr.logger.Infof(ctx, "finding page %d of users with country '%s' created after '%s'", query.Page, query.Country, query.CreatedAfter)

	page, err := svr.service.FindUsers(ctx, &user.Query{
		CreatedAfter: query.CreatedAfter,
		Country:      query.Country,
		Length:       query.Length,
		Page:         query.Page,
	})
	if err != nil {
		span.RecordError(err)
		svr.logger.Errorf(ctx, err, "error finding page %d of users with country '%s' created after '%s'", query.Page, query.Country, query.CreatedAfter)
		return nil, status.Error(codes.Internal, msgInternalServerError)
	}
	return pbPageFromPage(&page), nil
}
