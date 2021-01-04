define docker_run
	docker run --name venom-$(2) -d $(3) $(1) > venom-$2.cid
endef

docker-network := venom-test-net

venom-postgres.cid:
	$(call docker_run,postgres,postgres,-p 15432:5432 -e POSTGRES_PASSWORD=venom -e POSTGRES_USER=venom -e POSTGRES_DB=venom)
venom-mysql.cid:
	$(call docker_run,mysql/mysql-server,mysql,-p 13306:3306 -e MYSQL_PASSWORD=venom -e MYSQL_USER=venom -e MYSQL_DATABASE=venom)
venom-redis.cid:
	$(call docker_run,redis,redis,-p 16379:6379)
venom-imap.cid:
	$(call docker_run,antespi/docker-imap-devel,imap,-p 1025:25 -p 1143:143 -p 1993:993 -e MAILNAME=example.org -e MAIL_ADDRESS=address@example.org -e MAIL_PASS=pass)
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
	$(call docker_run,ghcr.io/linuxserver/openssh-server,sshd,-p 2222:2222 -e PUID=1000 -e PGID=1000 -e TZ=Europe/London -e PUBLIC_KEY="$(shell cat ~/.ssh/id_rsa.pub)" -e USER_NAME=venom )

start-test-stack: venom-postgres.cid venom-mysql.cid venom-redis.cid venom-imap.cid venom-kafka.cid venom-rabbit.cid venom-sshd.cid

stop-test-stack:
	@for f in `ls -1 *.cid`; do docker stop `cat $${f}`; docker rm `cat $${f}`; done; rm -f *.cid; docker network rm $(docker-network)

run-test: wait-for-kafka
	@(cd ../cmd/venom; go build && VENOM_VAR_MY_ENVAR=foo ./venom run --stop-on-failure -vv --var-from-file ../../tests/kafka/testVariables.yml  --var-from-file ../../tests/vars/vars.yml ../../tests/*.yml ../../tests/assertions/*.yml)

.PHONY: wait-for-kafka
wait-for-kafka:
	@printf "waiting for Kafka infrastructure startup ... (it may take a few minutes)" ; \
	until curl --retry 30 --retry-connrefused --retry-max-time 100 --connect-timeout 10 -s http://localhost:8081/config/ -o /dev/null; do sleep 2; done ; \
	echo "\n\033[0;32mdone\033[0m"

clean:
	@rm -f *.prof *.log *.dump.json
