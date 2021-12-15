package gherkin

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors/amqp"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/grpc"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/kafka"
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

var (
	path []string
	v    *venom.GherkinVenom

	variables     []string
	format        string
	varFiles      []string
	outputDir     string
	libDir        string
	stopOnFailure bool
	verbose       int
)

// Cmd run
var Cmd = &cobra.Command{
	Use:   "gherkin",
	Short: "Run Gherkin scenarios",
	Long: `
$ venom gherkin *.feature`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			path = append(path, ".")
		} else {
			path = args[0:]
		}

		v = venom.NewGherkin()
		v.RegisterExecutorBuiltin(amqp.Name, amqp.New())
		v.RegisterExecutorBuiltin(dbfixtures.Name, dbfixtures.New())
		v.RegisterExecutorBuiltin(exec.Name, exec.New())
		v.RegisterExecutorBuiltin(grpc.Name, grpc.New())
		v.RegisterExecutorBuiltin(http.Name, http.New())
		v.RegisterExecutorBuiltin(imap.Name, imap.New())
		v.RegisterExecutorBuiltin(kafka.Name, kafka.New())
		v.RegisterExecutorBuiltin(mqtt.Name, mqtt.New())
		v.RegisterExecutorBuiltin(ovhapi.Name, ovhapi.New())
		v.RegisterExecutorBuiltin(rabbitmq.Name, rabbitmq.New())
		v.RegisterExecutorBuiltin(readfile.Name, readfile.New())
		v.RegisterExecutorBuiltin(redis.Name, redis.New())
		v.RegisterExecutorBuiltin(smtp.Name, smtp.New())
		v.RegisterExecutorBuiltin(sql.Name, sql.New())
		v.RegisterExecutorBuiltin(ssh.Name, ssh.New())
		v.RegisterExecutorBuiltin(web.Name, web.New())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		v.Verbose = 3
		if err := v.InitLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		err := v.ParseGherkin(context.Background(), path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(2)
			return err
		}

		return nil
	},
}
