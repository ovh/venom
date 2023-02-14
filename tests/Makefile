PKI_DIR := ./grpc/pki
PKI_VAR_FILE = ./pki_variables.yml

include ./pki.mk

define docker_run
	docker run --name venom-$(2) -d $(3) $(1) > venom-$2.cid
endef

docker-network := venom-test-net

COVER_FILES = $(shell find . -name "*.coverprofile")
PKGS_COMMA_SEP = go list -f '{{ join .Deps "\n" }}{{"\n"}}{{.ImportPath}}' . | grep github.com/ovh/venom | grep -v vendor | tr '\n' ',' | sed 's/,$$//'

venom-postgres.cid:
	$(call docker_run,postgres,postgres,-p 15432:5432 -e POSTGRES_PASSWORD=venom -e POSTGRES_USER=venom -e POSTGRES_DB=venom)
venom-mysql.cid:
	$(call docker_run,mysql/mysql-server,mysql,-p 13306:3306 -e MYSQL_PASSWORD=venom -e MYSQL_USER=venom -e MYSQL_DATABASE=venom)
venom-mongo.cid:
	$(call docker_run,mongo,mongo,-p 27017:27017 -e MONGO_INITDB_ROOT_PASSWORD=venom -e MONGO_INITDB_ROOT_USERNAME=venom -e MONGO_INITDB_DATABASE=venom)
venom-redis.cid:
	$(call docker_run,redis,redis,-p 16379:6379)
venom-imap.cid:
	$(call docker_run,tomadele/mail-server,imap,-p 1025:25 -p 1143:143 -p 1993:993 -e MAILNAME=example.org -e MAIL_ADDRESS=address@example.org -e MAIL_PASS=pass)
venom-kafka.cid:
	docker network create --driver bridge $(docker-network)
	$(call docker_run,confluentinc/cp-zookeeper:6.0.0,kafka-zookeeper,\
		--network $(docker-network) \
		-p 2181:2181 \
		-e ZOOKEEPER_CLIENT_PORT=2181 \
		-e ZOOKEEPER_TICK_TIME=2000 \
	)
	$(call docker_run,confluentinc/cp-kafka:6.0.0,kafka-broker,\
		--network $(docker-network) \
		-p 9092:9092 -p 9101:9101 -p 29092:29092 \
		--env-file kafka/broker.env \
	)
	$(call docker_run,confluentinc/cp-schema-registry:6.0.0,kafka-schema-registry,\
		--network $(docker-network) \
		-p 8081:8081 \
		-e SCHEMA_REGISTRY_HOST_NAME=venom-kafka-schema-registry \
		-e SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS=venom-kafka-broker:29092 \
	)
venom-rabbit.cid:
	$(call docker_run,rabbitmq,rabbitmq,-p 5672:5672 -p 15672:15672)
venom-sshd.cid:
	$(call docker_run,ghcr.io/linuxserver/openssh-server,sshd,-p 2222:2222 -e PUID=1000 -e PGID=1000 -e TZ=Europe/London -e PUBLIC_KEY="$(shell cat ~/.ssh/id_rsa.pub)" -e USER_NAME=venom -e PASSWORD_ACCESS=true -e USER_PASSWORD=testvenom -e SUDO_ACCESS=true)
venom-mqtt.cid:
	$(call docker_run,eclipse-mosquitto,mqtt-broker,-p 1883:1883 -p 9001:9001 -v $(shell realpath mqtt/mosquitto.conf):/mosquitto/config/mosquitto.conf:ro)
venom-qpid.cid:
	$(call docker_run,scholzj/qpid-cpp:1.39.0,qpid,-p 5673:5672)
	docker exec venom-qpid qpid-config add queue amqp-test
venom-grpc.cid:
	docker build -t venom-grpc-greeter ./grpc
	$(call docker_run,venom-grpc-greeter,grpc,-p 50051:50051)

start-test-stack: venom-postgres.cid
start-test-stack: venom-mysql.cid
start-test-stack: venom-mongo.cid
start-test-stack: venom-redis.cid
start-test-stack: venom-imap.cid
start-test-stack: venom-kafka.cid
start-test-stack: venom-rabbit.cid
start-test-stack: venom-sshd.cid
start-test-stack: venom-mqtt.cid
start-test-stack: venom-qpid.cid
start-test-stack: $(PKI_DIR)
start-test-stack: venom-grpc.cid

stop-test-stack:
	@for f in `ls -1 *.cid`; do docker stop `cat $${f}`; docker rm `cat $${f}`; done; rm -f *.cid; docker network rm $(docker-network)

build-test-binary:
	cd ../cmd/venom; \
	TEMP=`$(PKGS_COMMA_SEP)`; \
	CGO_ENABLED=1 go test -coverpkg $$TEMP -c . -o ../../tests/venom.test -ldflags "-X github.com/ovh/venom.IsTest=true" -tags testbincover;

build-test-binary-docker:
	docker run -v `pwd`/..:/workspace golang:1.19 sh -c "cd /workspace/tests && make build-test-binary"	

run-test: generate-venom-pki
	VENOM_VAR_MY_ENVAR=foo  ./venom_wrapper.sh run \
	-vv --format=xml --output-dir=. --html-report --lib-dir=./lib_custom --var='array_from_var=["biz","buz"]' --var-from-file ./kafka/testVariables.yml --var-from-file=$(PKI_VAR_FILE) --var-from-file ./vars/vars.yml ./*.yml ./assertions/*.yml && \
	(cd ./vars_override && VENOM_VAR_foo=from-env VENOM_VAR_FOO2=from-env2 VENOM_VAR_FOO3=from-env3 ../venom_wrapper.sh run -vv --format=xml --output-dir=. --var foo='from-cmd-arg' --var='array_from_var=["biz","buz"]' ./mytc.yml);

.PHONY: wait-for-kafka
wait-for-kafka:
	@printf "waiting for Kafka infrastructure startup ... (it may take a few minutes)" ; \
	until curl --retry 30 --retry-connrefused --retry-max-time 100 --connect-timeout 10 -s http://localhost:8081/config/ -o /dev/null; do sleep 2; done ; \
	echo "\n\033[0;32mdone\033[0m"

clean:
	@rm -f *.prof *.html *.xml *.log *.dump.json *.args.file *.error.out *.out *.test.out *.coverprofile

merge-coverage:
	@docker run -v `pwd`:/workspace golang:1.19 sh -c "\
			go install github.com/wadey/gocovmerge@latest && \
			cd /workspace && \
    gocovmerge $(COVER_FILES) > /workspace/venom.cover.out \
		"
