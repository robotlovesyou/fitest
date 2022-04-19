package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/event"
	"github.com/robotlovesyou/fitest/pkg/health"
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
	"google.golang.org/grpc/reflection"
)

const (
	RpcPortVar     = "RPC_PORT"
	HealthPortVar  = "HEALTH_PORT"
	DatabaseURIVar = "DATABASE_URI"
	JaegerURIVar   = "JAEGER_URI"

	// DatabaseConnectionTimeout is the time allowed to make an initial connection to the database.
	// It should be configurable
	DatabaseConnectionTimeout = 30 * time.Second

	//Interface Addr is the interface to listen on. It should probably be configurable
	InterfaceAddr = "0.0.0.0"
	//HealthcheckPath is the path for the healthcheck.
	HealthcheckPath = "/healthy"
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

func createStore() (*userstore.Store, error) {
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
	db := client.Database(strings.TrimLeft(uri.Path, "/"))
	store := userstore.New(db)
	err = store.EnsureIndexes(ctx) // This should not really be done at service startup
	if err != nil {
		return nil, fmt.Errorf("cannot create indexes: %w", err)
	}

	return store, nil
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
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", InterfaceAddr, port))
	if err != nil {
		return nil, fmt.Errorf("canoot bind to port %d, %w", port, err)
	}
	stdlog.Printf("RPC listening on %s:%d", InterfaceAddr, port)
	grpcServer := grpc.NewServer()
	userspb.RegisterUsersServer(grpcServer, rpc.New(service, logger))
	reflection.Register(grpcServer)
	go grpcServer.Serve(lis)

	return grpcServer, nil
}

func startpublishingChanges(ctx context.Context, service *user.Service) {
	go service.PublishChanges(ctx)
}

func startHealthcheck(logger *log.Logger, store *userstore.Store, service *user.Service) (*http.Server, error) {
	port, err := healthcheckPort()
	if err != nil {
		return nil, err
	}
	svc := health.New(logger, userstore.NewMonitor(store), user.NewMonitor(service))
	mux := http.NewServeMux()
	mux.HandleFunc(HealthcheckPath, svc.Handle)
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", InterfaceAddr, port),
		Handler: mux,
	}
	go func() {
		stdlog.Printf("healtcheck starting on %s", server.Addr)
		err := server.ListenAndServe()
		stdlog.Printf("healthcheck server has exited: %v", err)
	}()
	return server, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	store, err := createStore()
	if err != nil {
		stdlog.Fatal(err)
	}

	logger, err := createLogger()
	if err != nil {
		stdlog.Fatal(err)
	}

	service := createUserService(store, createEventBus(), logger)

	rpcServer, err := startRPC(service, logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	startpublishingChanges(ctx, service)

	healthServer, err := startHealthcheck(logger, store, service)
	if err != nil {
		stdlog.Fatal(err)
	}

	<-waitForExitSignal()
	rpcServer.Stop()
	healthServer.Close()
	cancel()

}
