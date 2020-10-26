


TEST_REDIS_PORT =  $(if ${VENOM_TEST_REDIS_PORT},${VENOM_TEST_REDIS_PORT},6379)
TEST_REDIS_START = docker run -d -p $(TEST_REDIS_PORT):6379 --name redis-venom registry.ovh.net/official/redis:3.2.5

TEST_PG_PORT = $(if ${VENOM_TEST_PG_PORT},${VENOM_TEST_PG_PORT},5432)
TEST_PG_START = docker run -d -p $(TEST_PG_PORT):5432 -e POSTGRES_PASSWORD=venom -e POSTGRES_USER=venom -e POSTGRES_DB=venom --name pg-venom postgres

TEST_MYSQL_PORT =  $(if ${VENOM_TEST_MYSQL_PORT},${VENOM_TEST_MYSQL_PORT},3306)
TEST_MYSQL_START = docker run -d -p $(TEST_MYSQL_PORT):3306 -e MYSQL_PASSWORD=venom -e MYSQL_USER=venom -e MYSQL_DATABASE=venom --name mysql-venom mysql/mysql-server


TEST_REDIS_HOST =  $(if ${VENOM_TEST_REDIS_HOST},${VENOM_TEST_REDIS_HOST},localhost)
TEST_MYSQL_HOST =  $(if ${VENOM_TEST_MYSQL_HOST},${VENOM_TEST_MYSQL_HOST},localhost)
TEST_PG_HOST = $(if ${VENOM_TEST_PG_HOST},${VENOM_TEST_PG_HOST},localhost)



test-docker-start:
	$(TEST_REDIS_START)
	$(TEST_PG_START)
	$(TEST_MYSQL_START)
	@sleep 15

clean:
	docker stop redis-venom; docker rm redis-venom; exit 0
	docker stop mysql-venom; docker rm mysql-venom; exit 0
	docker stop pg-venom; docker rm pg-venom; exit 0
	docker stop smtp-venom; docker rm smtp-venom; exit 0


test: clean test-docker-start
	venom run . \
		--var redisHost=localhost \
		--var smtpPort=${TEST__SMTP_PORT} \
		--var redisPort=${TEST_REDIS_PORT} \
		--var mysqlHost=localhost \
		--var mysqlPort=${TEST_MYSQL_PORT} \
		--var pgHost=localhost \
		--var pgPort=${TEST_PG_PORT} \
		--exclude MyTestSuiteWeb.yml \
		--exclude MyTestSuiteSMTP.yml \
		--exclude MyTestSuiteImap.yml \
		--exclude MyTestSuiteSSH.yml \
		* \
		--details high

.PHONY: local-env-start
local-env-start:
	$(DOCK_COMP_CMD) up \
		--force-recreate \
		--remove-orphans \
		zookeeper kafka0 schema-registry || (docker ps -a -q | xargs docker rm -vf ; exit 1)

.PHONY: local-env-stop
local-env-stop:
	$(DOCK_COMP_CMD) down --volumes --remove-orphans
	docker image prune --force --filter label=stage=test
	docker images -aq -f 'dangling=true' | xargs docker rmi

DOCK_COMP_CMD = docker-compose -f kafka/docker-compose.yml

.PHONY: test-kafka
test-kafka: build-venom-image
	$(eval VENOM_CONTAINER_NAME = venom-container-$(shell date +%s))

	go mod download
	go mod vendor
	$(DOCK_COMP_CMD) run \
		--detach \
		--name $(VENOM_CONTAINER_NAME) \
		venom tail -f /dev/null

	docker exec $(VENOM_CONTAINER_NAME) venom run \
		--log debug \
		--parallel 5 \
		--format xml \
		--output-dir "." \
		--strict \
		tests/MyTestSuiteKafka.yml || true

	mkdir results
	docker exec $(VENOM_CONTAINER_NAME) cat test_results.xml > results/test_results.xml ; \
	docker exec $(VENOM_CONTAINER_NAME) cat my-kafka-testsuite.kafka-test.dump > results/my-kafka-testsuite.kafka-test.dump ; \
	$(DOCK_COMP_CMD) rm --force --stop -v venom || true

.PHONY: build-venom-image
build-venom-image:
	$(DOCK_COMP_CMD) build \
		--force-rm \
		venom || \
		(docker ps -a -q | xargs docker rm -vf; exit 1)
