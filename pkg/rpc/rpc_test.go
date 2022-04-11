package rpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pb"
	"github.com/robotlovesyou/fitest/pkg/rpc"
	"github.com/robotlovesyou/fitest/pkg/users"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type stubCreateUser func(context.Context, users.NewUser) (users.User, error)

type stubUsersService struct {
	createUser stubCreateUser
}

func newStubService() *stubUsersService {
	return &stubUsersService{
		createUser: func(context.Context, users.NewUser) (users.User, error) {
			panic("stub create user")
		},
	}
}

func (svc *stubUsersService) CreateUser(ctx context.Context, newUser users.NewUser) (users.User, error) {
	return svc.createUser(ctx, newUser)
}

func fakeNewUser() pb.NewUser {
	password := faker.Password()
	return pb.NewUser{
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Nickname:        faker.Username(),
		Password:        password,
		ConfirmPassword: password,
		Email:           faker.Email(),
		Country:         "DE",
	}
}

func userFromNewUser(newUser users.NewUser) users.User {
	return users.User{
		ID:           uuid.Must(uuid.NewRandom()),
		FirstName:    newUser.FirstName,
		LastName:     newUser.LastName,
		Nickname:     newUser.Nickname,
		PasswordHash: "HashOfASuperSecretPassword",
		Email:        newUser.Email,
		Country:      newUser.Country,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// withClient creates and instantiates a grpc server which delegates calls to the provided
// rpc.UsersService imlementation, and calls the callback f with a client connected to the
// grpc server
func withClient(svc rpc.UsersService, f func(pb.UsersClient)) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("cannot open random port: %v", err))
	}
	serverAddress := lis.Addr().String()

	grpcServer := grpc.NewServer()
	pb.RegisterUsersServer(grpcServer, rpc.New(svc))
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("cannot dial rpc server: %v", err))
	}
	defer conn.Close()
	client := pb.NewUsersClient(conn)
	f(client)
}

func TestCreateUserRPCCallsUsersServiceWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	request := fakeNewUser()
	var response users.User
	withClient(stubService, func(client pb.UsersClient) {
		// check that the request payload has been conveyed correctly to the users service
		stubService.createUser = func(ctx context.Context, newUser users.NewUser) (users.User, error) {
			require.Equal(t, request.FirstName, newUser.FirstName)
			require.Equal(t, request.LastName, newUser.LastName)
			require.Equal(t, request.Nickname, newUser.Nickname)
			require.Equal(t, request.Password, newUser.Password)
			require.Equal(t, request.ConfirmPassword, newUser.ConfirmPassword)
			require.Equal(t, request.Email, newUser.Email)
			require.Equal(t, request.Country, newUser.Country)
			response = userFromNewUser(newUser)
			return response, nil
		}

		// check that the created user has been conveyed correctly via the rpc layer
		user, err := client.CreateUser(context.Background(), &request)
		require.NoError(t, err)
		require.Equal(t, response.ID.String(), user.Id)
		require.Equal(t, response.FirstName, user.FirstName)
		require.Equal(t, response.LastName, user.LastName)
		require.Equal(t, response.Nickname, user.Nickname)
		require.Equal(t, response.Email, user.Email)
		require.Equal(t, response.Country, user.Country)
		require.Equal(t, response.CreatedAt.Format(time.RFC3339), user.CreatedAt)
		require.Equal(t, response.UpdatedAt.Format(time.RFC3339), user.UpdatedAt)
	})
}
