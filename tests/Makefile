


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
	venom run . --var redisHost=localhost --var smtpPort=${TEST__SMTP_PORT} --var redisPort=${TEST_REDIS_PORT} --var mysqlHost=localhost --var mysqlPort=${TEST_MYSQL_PORT} --var pgHost=localhost --var pgPort=${TEST_PG_PORT} --exclude MyTestSuiteWeb.yml --exclude MyTestSuiteSMTP.yml --exclude MyTestSuiteImap.yml --exclude MyTestSuiteSSH.yml * --details high
