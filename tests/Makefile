define docker_run
	docker run --name venom-$(2) -d $(3) $(1) > venom-$2.cid
endef

venom-postgres.cid:
	$(call docker_run,postgres,postgres,-p 15432:5432 -e POSTGRES_PASSWORD=venom -e POSTGRES_USER=venom -e POSTGRES_DB=venom)
venom-mysql.cid:
	$(call docker_run,mysql/mysql-server,mysql,-p 13306:3306 -e MYSQL_PASSWORD=venom -e MYSQL_USER=venom -e MYSQL_DATABASE=venom)
venom-redis.cid:
	$(call docker_run,redis,redis,-p 16379:6379)
venom-imap.cid:
	$(call docker_run,antespi/docker-imap-devel,imap,-p 1025:25 -p 1143:143 -p 1993:993 -e MAILNAME=example.org -e MAIL_ADDRESS=address@example.org -e MAIL_PASS=pass)
venom-kafka.cid:
	$(call docker_run,spotify/kafka,kafka,-p 2181:2181 -p 9092:9092 -e ADVERTISED_HOST=$(shell hostname) -e ADVERTISED_PORT=9092)
venom-rabbit.cid:
	$(call docker_run,rabbitmq,rabbitmq,-p 5672:5672 -p 15672:15672)
venom-sshd.cid:
	$(call docker_run,ghcr.io/linuxserver/openssh-server,sshd,-p 2222:2222 -e PUID=1000 -e PGID=1000 -e TZ=Europe/London -e PUBLIC_KEY="$(shell cat ~/.ssh/id_rsa.pub)" -e USER_NAME=venom )

start-test-stack: venom-postgres.cid venom-mysql.cid venom-redis.cid venom-imap.cid venom-kafka.cid venom-rabbit.cid venom-sshd.cid

stop-test-stack:
	@for f in `ls -1 *.cid`; do docker stop `cat $${f}`; docker rm `cat $${f}`; done; rm -f *.cid

run-test: 
	@(cd ../cmd/venom; go build && ./venom run --stop-on-failure  --log debug ../../tests/*.yml)


