package executors

import (
	"github.com/ovh/venom"
	"github.com/ovh/venom/executors/amqp"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/grpc"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/kafka"
	"github.com/ovh/venom/executors/mongo"
	"github.com/ovh/venom/executors/mqtt"
	"github.com/ovh/venom/executors/ovhapi"
	"github.com/ovh/venom/executors/rabbitmq"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/redis"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/sql"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
)

type Constructor func() venom.Executor

// Registry is a map of executors to executor constructor functions.
var Registry map[string]Constructor = map[string]Constructor{
	amqp.Name:       amqp.New,
	dbfixtures.Name: dbfixtures.New,
	exec.Name:       exec.New,
	grpc.Name:       grpc.New,
	http.Name:       http.New,
	imap.Name:       imap.New,
	kafka.Name:      kafka.New,
	mqtt.Name:       mqtt.New,
	ovhapi.Name:     ovhapi.New,
	rabbitmq.Name:   rabbitmq.New,
	readfile.Name:   readfile.New,
	redis.Name:      redis.New,
	smtp.Name:       smtp.New,
	sql.Name:        sql.New,
	ssh.Name:        ssh.New,
	mongo.Name:      mongo.New,
	web.Name:        web.New,
}
