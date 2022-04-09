TEST = go test ./... -count=1

test:
	$(TEST)

test_cover:
	$(TEST) -coverpkg=./... -coverprofile cp.out && go tool cover -func cp.out && rm cp.out

test_for_ci:
	$(TEST) -race -v

lint:
	staticcheck ./...
