PACKAGES = github.com/robotlovesyou/fitest/pkg/... github.com/robotlovesyou/fitest/cmd/...
TEST = go test $(PACKAGES) -count=1

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
	cd pb && \
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    users.proto && \
	popd
