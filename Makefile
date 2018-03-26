GOPATH ?= $(shell go env GOPATH)
BINARIES = rebump
API_FILES = api/rebump.pb.go api/rebump.pb.gw.go api/rebump.swagger.json

.PHONY: build api dep test race msan

build: api dep ## Build rebump
	@go build .

dep: api ## Fetch dependencies
	@go get ./...

api: $(API_FILES) ## Auto-generate gRPC/REST Go sources

api/rebump.pb.go: api/rebump.proto
	@protoc -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:. \
		api/rebump.proto

api/rebump.pb.gw.go: api/rebump.proto
	@protoc -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:. \
		api/rebump.proto

api/rebump.swagger.json: api/rebump.proto
	@protoc -I. \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--swagger_out=logtostderr=true:. \
		api/rebump.proto

clean: ## Clean compiled binaries
	@rm -f ${BINARIES}

realclean: ## Clean compiled binaries and all generated files
	@rm -f ${BINARIES}
	@rm -f ${API_FILES}

test: dep ## Run tests
	@go test ./...

race: dep ## Run tests with race detector
	@go test -race ./...

msan: dep ## Run tests with memory sanitizer
	@go test -msan ./...

lint: ## Run linter on all go sources
	@golint -set_exit_status ./...