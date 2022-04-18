package rpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/bxcodec/faker/v3"
	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/robotlovesyou/fitest/pkg/rpc"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/robotlovesyou/fitest/pkg/utctime"
	"github.com/robotlovesyou/fitest/userspb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Hand coded stub/mock for Users service
//// I prefer hand coded stubs where appropriate because the code created by
//// mockgen makes me sad!
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type stubCreate func(context.Context, *user.NewUser) (user.User, error)
type stubUpdate func(context.Context, *user.Update) (user.User, error)
type stubDelete func(context.Context, *user.Ref) error
type stubFind func(context.Context, *user.Query) (user.Page, error)

type stubUsersService struct {
	create stubCreate
	update stubUpdate
	delete stubDelete
	find   stubFind
}

func newStubService() *stubUsersService {
	return &stubUsersService{
		create: func(context.Context, *user.NewUser) (user.User, error) {
			panic("stub create user")
		},
		update: func(context.Context, *user.Update) (user.User, error) {
			panic("stub update user")
		},
		delete: func(context.Context, *user.Ref) error {
			panic("stub delete user")
		},
		find: func(context.Context, *user.Query) (user.Page, error) {
			panic("stub find users")
		},
	}
}

func (svc *stubUsersService) Create(ctx context.Context, newUser *user.NewUser) (user.User, error) {
	return svc.create(ctx, newUser)
}

func (svc *stubUsersService) Update(ctx context.Context, userUpdate *user.Update) (user.User, error) {
	return svc.update(ctx, userUpdate)
}

func (svc *stubUsersService) Delete(ctx context.Context, userRef *user.Ref) error {
	return svc.delete(ctx, userRef)
}

func (svc stubUsersService) Find(ctx context.Context, query *user.Query) (user.Page, error) {
	return svc.find(ctx, query)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Test helper functions
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

// fakeNewUser creates new users using faker for testing
func fakeNewUser() userspb.NewUser {
	password := faker.Password()
	return userspb.NewUser{
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Nickname:        faker.Username(),
		Password:        password,
		ConfirmPassword: password,
		Email:           faker.Email(),
		Country:         "DE",
	}
}

// fakeUserUpdate creates a fake user update using faker for testing
func fakeUserUpdate() userspb.Update {
	password := faker.Password()
	return userspb.Update{
		Id:              uuid.Must(uuid.NewRandom()).String(),
		FirstName:       faker.FirstName(),
		LastName:        faker.LastName(),
		Password:        password,
		ConfirmPassword: password,
		Country:         "DE",
		Version:         0,
	}
}

// fakeUserRef creates a fake user ref for testing
func fakeUserRef() userspb.Ref {
	return userspb.Ref{
		Id: uuid.Must(uuid.NewRandom()).String(),
	}
}

// fakeUsersQuery creates a fake query for testing
func fakeUsersQuery() userspb.Query {
	return userspb.Query{
		CreatedAfter: utctime.Now().Format(user.TimeFormat),
		Country:      "DE",
		Length:       10,
		Page:         11,
	}
}

// fake user creates a fake user for testing
func fakeSanitizedUser() user.SanitizedUser {
	return user.SanitizedUser{
		ID:        uuid.Must(uuid.NewRandom()).String(),
		FirstName: faker.FirstName(),
		LastName:  faker.LastName(),
		Nickname:  faker.Username(),
		Email:     faker.Email(),
		Country:   "DE",
		CreatedAt: utctime.Now().Format(user.TimeFormat),
		UpdatedAt: utctime.Now().Format(user.TimeFormat),
	}
}

// userFromNewUser creates a fake user from a new user for testing
func userFromNewUser(newUser user.NewUser) user.User {
	return user.User{
		ID:           uuid.Must(uuid.NewRandom()),
		FirstName:    newUser.FirstName,
		LastName:     newUser.LastName,
		Nickname:     newUser.Nickname,
		PasswordHash: "HashOfASuperSecretPassword",
		Email:        newUser.Email,
		Country:      newUser.Country,
		CreatedAt:    utctime.Now(),
		UpdatedAt:    utctime.Now(),
	}
}

// userFromUserUpdate creates a fake user from a user update for testing
func userFromUserUpdate(userUpdate user.Update) user.User {
	return user.User{
		ID:           uuid.Must(uuid.NewRandom()),
		FirstName:    userUpdate.FirstName,
		LastName:     userUpdate.LastName,
		Email:        faker.Email(),
		Nickname:     faker.Username(),
		PasswordHash: "HashOfASuperSecretPassword",
		Country:      userUpdate.Country,
		CreatedAt:    utctime.Now(),
		UpdatedAt:    utctime.Now(),
	}
}

// usersPageFromQuery creates a page of fake users from a query for testing
func usersPageFromQuery(query user.Query) user.Page {
	items := make([]user.SanitizedUser, 0, query.Length)
	for i := 0; i < int(query.Length); i += 1 {
		items = append(items, fakeSanitizedUser())
	}
	return user.Page{
		Page:  query.Page,
		Total: query.Page * int64(query.Length),
		Items: items,
	}
}

// compareUserToPBUser compares a user.User to a userpb.User
func compareUserToPBUser(t *testing.T, usr user.User, pbUser *userspb.User) {
	require.Equal(t, usr.ID.String(), pbUser.Id)
	require.Equal(t, usr.FirstName, pbUser.FirstName)
	require.Equal(t, usr.LastName, pbUser.LastName)
	require.Equal(t, usr.Nickname, pbUser.Nickname)
	require.Equal(t, usr.Email, pbUser.Email)
	require.Equal(t, usr.Country, pbUser.Country)
	require.Equal(t, usr.CreatedAt.Format(user.TimeFormat), pbUser.CreatedAt)
	require.Equal(t, usr.UpdatedAt.Format(user.TimeFormat), pbUser.UpdatedAt)
}

func compareSanitizedUserToPBUser(t *testing.T, usr user.SanitizedUser, pbUser *userspb.User) {
	require.Equal(t, usr.ID, pbUser.Id)
	require.Equal(t, usr.FirstName, pbUser.FirstName)
	require.Equal(t, usr.LastName, pbUser.LastName)
	require.Equal(t, usr.Nickname, pbUser.Nickname)
	require.Equal(t, usr.Email, pbUser.Email)
	require.Equal(t, usr.Country, pbUser.Country)
	require.Equal(t, usr.CreatedAt, pbUser.CreatedAt)
	require.Equal(t, usr.UpdatedAt, pbUser.UpdatedAt)
}

// withClient creates and instantiates a grpc server which delegates calls to the provided
// rpc.UsersService imlementation, and calls the callback f with a client connected to the
// grpc server
func withClient(svc rpc.UsersService, f func(userspb.UsersClient)) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("cannot open random port: %v", err))
	}
	serverAddress := lis.Addr().String()

	logger, err := log.New("RPC Tests")
	if err != nil {
		panic("cannot create logger")
	}
	grpcServer := grpc.NewServer()
	userspb.RegisterUsersServer(grpcServer, rpc.New(svc, logger))
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("cannot dial rpc server: %v", err))
	}
	defer conn.Close()
	client := userspb.NewUsersClient(conn)
	f(client)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Tests
//// Happy path tests work by mocking the appropriate UserService method
//// and checking for a valid call, and then checking for a valid response from the rpc
////
//// Sad path tests stub for various error responses and ensure the correct grpc error
//// code is received by the client.
//// For a real world implementation I would also provide rich data where appropriate using
//// grpc status.Details
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func TestCreateUserRPCCallsUsersServiceWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	request := fakeNewUser()
	var response user.User
	withClient(stubService, func(client userspb.UsersClient) {
		// check that the request payload has been conveyed correctly to the users service
		stubService.create = func(ctx context.Context, newUser *user.NewUser) (user.User, error) {
			require.Equal(t, request.FirstName, newUser.FirstName)
			require.Equal(t, request.LastName, newUser.LastName)
			require.Equal(t, request.Nickname, newUser.Nickname)
			require.Equal(t, request.Password, newUser.Password)
			require.Equal(t, request.ConfirmPassword, newUser.ConfirmPassword)
			require.Equal(t, request.Email, newUser.Email)
			require.Equal(t, request.Country, newUser.Country)
			response = userFromNewUser(*newUser)
			return response, nil
		}

		// check that the created user has been conveyed correctly via the rpc layer
		user, err := client.CreateUser(context.Background(), &request)
		require.NoError(t, err)
		compareUserToPBUser(t, response, user)
	})
}

func TestCorrectErrorCodesSentCreatingUser(t *testing.T) {
	// For the sake of brevity, I am only going to use grpc error codes when the service fails.
	// In a real world implementation I would, where appropriate, include detail via status details
	cases := []struct {
		name         string
		result       error
		expectedCode codes.Code
	}{
		{
			name:         "Already exists",
			result:       user.ErrAlreadyExists,
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "Invalid",
			result:       user.ErrInvalid,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "Internal",
			result:       errors.New("some unexpected error"),
			expectedCode: codes.Internal,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stubService := newStubService()
			request := fakeNewUser()
			withClient(stubService, func(client userspb.UsersClient) {
				stubService.create = func(ctx context.Context, _ *user.NewUser) (usr user.User, err error) {
					return usr, testCase.result
				}

				_, err := client.CreateUser(context.Background(), &request)
				require.Equal(t, testCase.expectedCode.String(), status.Code(err).String())
			})
		})
	}
}

func TestUpdateUserRPCCallsServiceAndRespondsWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	request := fakeUserUpdate()
	var response user.User
	withClient(stubService, func(client userspb.UsersClient) {
		// check that the request payload has been conveyed correctly to the users service
		stubService.update = func(ctx context.Context, userUpdate *user.Update) (user.User, error) {
			require.Equal(t, request.Id, userUpdate.ID)
			require.Equal(t, request.FirstName, userUpdate.FirstName)
			require.Equal(t, request.LastName, userUpdate.LastName)
			require.Equal(t, request.Password, userUpdate.Password)
			require.Equal(t, request.ConfirmPassword, userUpdate.ConfirmPassword)

			require.Equal(t, request.Country, userUpdate.Country)
			response = userFromUserUpdate(*userUpdate)
			return response, nil
		}

		// check that the updated user has been conveyed correctly via the rpc layer
		user, err := client.UpdateUser(context.Background(), &request)
		require.NoError(t, err)
		compareUserToPBUser(t, response, user)
	})
}

func TestCorrectErrorCodesSentUpdatingUser(t *testing.T) {
	// For the sake of brevity, I am only going to use grpc error codes when the service fails.
	// In a real world implementation I would, where appropriate, include detail via status details
	cases := []struct {
		name         string
		result       error
		expectedCode codes.Code
	}{
		{
			name:         "NotFound",
			result:       user.ErrNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "Invalid",
			result:       user.ErrInvalid,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "InvalidVersion",
			result:       user.ErrInvalidVersion,
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "Internal",
			result:       errors.New("some unexpected error"),
			expectedCode: codes.Internal,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stubService := newStubService()
			request := fakeUserUpdate()
			withClient(stubService, func(client userspb.UsersClient) {
				stubService.update = func(ctx context.Context, _ *user.Update) (usr user.User, err error) {
					return usr, testCase.result
				}

				_, err := client.UpdateUser(context.Background(), &request)
				require.Equal(t, testCase.expectedCode.String(), status.Code(err).String())
			})
		})
	}
}

func TestDeleteUserRPCCallsUsersServiceAndRespondsWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	request := fakeUserRef()
	withClient(stubService, func(client userspb.UsersClient) {
		// check that the request payload has been conveyed correctly to the users service
		stubService.delete = func(ctx context.Context, ref *user.Ref) error {
			require.Equal(t, request.Id, ref.ID)
			return nil
		}

		_, err := client.DeleteUser(context.Background(), &request)
		require.NoError(t, err)
	})
}

func TestCorrectErrorCodesSentDeletingUser(t *testing.T) {
	// For the sake of brevity, I am only going to use grpc error codes when the service fails.
	// In a real world implementation I would, where appropriate, include detail via status details
	cases := []struct {
		name         string
		result       error
		expectedCode codes.Code
	}{
		{
			name:         "NotFound",
			result:       user.ErrNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "Invalid", // invalid could be returned if the id is malformed and cannot be parsed as a UUID
			result:       user.ErrInvalid,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "Internal",
			result:       errors.New("some unexpected error"),
			expectedCode: codes.Internal,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stubService := newStubService()
			request := fakeUserRef()
			withClient(stubService, func(client userspb.UsersClient) {
				stubService.delete = func(ctx context.Context, _ *user.Ref) error {
					return testCase.result
				}

				_, err := client.DeleteUser(context.Background(), &request)
				require.Equal(t, testCase.expectedCode.String(), status.Code(err).String())
			})
		})
	}
}

func TestFindUsersRPCCallsServiceAndRespondsWithCorrectValues(t *testing.T) {
	stubService := newStubService()
	request := fakeUsersQuery()
	var response user.Page
	withClient(stubService, func(client userspb.UsersClient) {
		// check that the request payload has been conveyed correctly to the users service
		stubService.find = func(ctx context.Context, query *user.Query) (user.Page, error) {
			require.Equal(t, request.CreatedAfter, query.CreatedAfter)
			require.Equal(t, request.Country, query.Country)
			require.Equal(t, request.Page, query.Page)
			require.Equal(t, request.Length, query.Length)

			response = usersPageFromQuery(*query)
			return response, nil
		}

		// check that the Page has been correctly conveyed by the RPC
		page, err := client.FindUsers(context.Background(), &request)
		require.NoError(t, err)
		require.Len(t, page.Items, len(response.Items))
		require.Equal(t, page.Total, response.Total)
		for i, itm := range page.Items {
			compareSanitizedUserToPBUser(t, response.Items[i], itm)
		}
	})
}

func TestCorrectErrorCodeSentFindingUsers(t *testing.T) {
	// For the sake of brevity, I am only going to use grpc error codes when the service fails.
	// In a real world implementation I would, where appropriate, include detail via status details
	stubService := newStubService()
	request := fakeUsersQuery()
	withClient(stubService, func(client userspb.UsersClient) {
		stubService.find = func(ctx context.Context, _ *user.Query) (page user.Page, err error) {
			return page, errors.New("some unexpected error")
		}

		_, err := client.FindUsers(context.Background(), &request)
		require.Equal(t, codes.Internal.String(), status.Code(err).String())
	})
}
