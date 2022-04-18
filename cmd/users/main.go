package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/event"
	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/robotlovesyou/fitest/pkg/password"
	"github.com/robotlovesyou/fitest/pkg/rpc"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/robotlovesyou/fitest/pkg/validation"
	"github.com/robotlovesyou/fitest/userspb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

const (
	RpcPortVar     = "RPC_PORT"
	HealthPortVar  = "HEALTH_PORT"
	DatabaseURIVar = "DATABASE_URI"

	// DatabaseConnectionTimeout is the time allowed to make an initial connection to the database.
	// It should be configurable
	DatabaseConnectionTimeout = 30 * time.Second
)

func getEnvI32(name string) (int32, error) {
	port, err := strconv.ParseInt(os.Getenv(name), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %s: %w", name, err)
	}
	return int32(port), nil
}

func rpcPort() (int32, error) {
	return getEnvI32(RpcPortVar)
}

func healthcheckPort() (int32, error) {
	return getEnvI32(HealthPortVar)
}

func databaseURI() string {
	return os.Getenv(DatabaseURIVar)
}

func createStore() (user.UserStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DatabaseConnectionTimeout)
	defer cancel()

	uri, err := url.Parse(databaseURI())
	if err != nil {
		return nil, fmt.Errorf("cannot parse database conection uri: %w", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri.String()))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongo server: %w", err)
	}
	db := client.Database(uri.Path)
	return userstore.New(db), nil
}

func createEventBus() event.Bus {
	return event.New()
}

func createLogger() (*log.Logger, error) {
	logger, err := log.New("Users Service") // Service name could be configurable?
	if err != nil {
		return nil, fmt.Errorf("cannot create logger: %w", err)
	}
	return logger, nil
}

func createUserService(store user.UserStore, bus event.Bus, logger *log.Logger) *user.Service {
	return user.New(store, password.New(), uuid.NewRandom, validation.New(), bus, logger)
}

func waitForExitSignal() <-chan bool {
	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		stdlog.Printf("Received exit signal %v", sig)
		done <- true
	}()
	return done
}

func startRPC(service *user.Service, logger *log.Logger) (*grpc.Server, error) {
	port, err := rpcPort()
	if err != nil {
		return nil, err
	}

	// It might be better to make the interface configurable as well as the port
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, fmt.Errorf("canoot bind to port %d, %w", port, err)
	}
	grpcServer := grpc.NewServer()
	userspb.RegisterUsersServer(grpcServer, rpc.New(service, logger))
	go grpcServer.Serve(lis)

	return grpcServer, nil
}

func main() {
	_, cancel := context.WithCancel(context.Background())
	store, err := createStore()
	if err != nil {
		panic(err)
	}

	logger, err := createLogger()
	if err != nil {
		panic(err)
	}

	service := createUserService(store, createEventBus(), logger)

	rpcServer, err := startRPC(service, logger)
	if err != nil {
		panic(err)
	}

	// TODO: Publish change events
	<-waitForExitSignal()
	rpcServer.Stop()
	cancel()

}
