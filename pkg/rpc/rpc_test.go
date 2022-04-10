package rpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/robotlovesyou/fitest/pkg/rpc"
	"github.com/robotlovesyou/fitest/pkg/rpc/generated"
	"github.com/robotlovesyou/fitest/pkg/users"
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

// withClient creates and instantiates a grpc server which delegates calls to the provided
// rpc.UsersService imlementation, and calls the callback f with a client connected to the
// grpc server
func withClient(svc rpc.UsersService, f func(generated.UsersClient)) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("cannot open random port: %v", err))
	}
	serverAddress := lis.Addr().String()

	grpcServer := grpc.NewServer()
	generated.RegisterUsersServer(grpcServer, rpc.New(svc))
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("cannot dial rpc server: %v", err))
	}
	defer conn.Close()
	client := generated.NewUsersClient(conn)
	f(client)
}

func TestCreateUserRPCCallsUsersServiceWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	withClient(stubService, func(client generated.UsersClient) {
		_, err := client.CreateUser(context.Background(), &generated.NewUser{})
		panic(err)
	})
}
