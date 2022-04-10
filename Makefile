TEST = go test ./... -count=1

test:
	$(TEST)

test_cover:
	$(TEST) -coverpkg=./... -coverprofile cp.out && go tool cover -func cp.out && rm cp.out

test_for_ci:
	$(TEST) -race -v

lint:
	staticcheck ./...

protoc:
	pushd . && \
	cd pkg/rpc && \
	protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    users.proto && \
	popd
