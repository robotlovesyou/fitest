PACKAGES = github.com/robotlovesyou/fitest/pkg/... github.com/robotlovesyou/fitest/cmd/...
TEST = go test $(PACKAGES) -count=1
						 
export DATABASE_TEST_URI = mongodb://root:password@localhost:27017/

test:
	$(TEST)

test_cover:
	$(TEST) -coverprofile cp.out -coverpkg=$(PACKAGES) && go tool cover -func cp.out && rm cp.out

test_for_ci:
	$(TEST) -race -v

lint:
	staticcheck ./...

protoc:
	pushd . && \
	cd userspb && \
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    users.proto && \
	popd

cloc: 
	cloc . --not-match-f=\.pb\.go

run:
	@RPC_PORT=8080 \
	DATABASE_URI=${DATABASE_TEST_URI}users?authSource=admin \
	HEALTH_PORT=9090 go run github.com/robotlovesyou/fitest/cmd/users/.

install:
	go install ./...
