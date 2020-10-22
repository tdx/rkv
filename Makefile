rkvd:
	cd cmd/rkvd && go build

pb:
	protoc internal/rpc/v1/*.proto \
			--gogofast_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto,plugins=grpc:. \
			--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
			--proto_path=.
	protoc internal/rpc/raft/*.proto \
			--gogofast_out=Mgogoproto/gogo.proto=github.com/gogo/protobuf/proto,plugins=grpc:. \
			--proto_path=$$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf) \
			--proto_path=.

docker:
	$(eval GIT_TAG=$(shell git describe --tags --abbrev=0))
	docker build -t rkvd:$(GIT_TAG) -f deploy/Dockerfile .
