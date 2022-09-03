CONFIG_PATH=${HOME}/.proglog/

.PHONY: init
init:
	@mkdir -p ${CONFIG_PATH}

.PHONY: run
run:
	@rm -r server-temp
	@go run cmd/server/main.go

.PHONY: proto
proto:
	@protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.

$(CONFIG_PATH)/model.conf:
	echo model && \
	cp test/model.conf $(CONFIG_PATH)/model.conf

$(CONFIG_PATH)/policy.csv:
	echo policy && \
	cp test/policy.csv $(CONFIG_PATH)/policy.csv

.PHONY: test
# policyとmodelコマンドに依存
# -raceオプションを付けるとテストがこけるので外しておく
test: $(CONFIG_PATH)/policy.csv $(CONFIG_PATH)/model.conf 
	go test ./...

.PHONy: gencert
gencert:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server
	
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client
	
	mv *.pem *.csr ${CONFIG_PATH}

TAG ?=v0.0.1

.PHONY: build-docker
build-docker:
	docker build -t morningnightdream/distributed-sevice-with-go:$(TAG) .

.PHONY: push-docker
push-docker:
	docker push morningnightdream/distributed-sevice-with-go:$(TAG) .
