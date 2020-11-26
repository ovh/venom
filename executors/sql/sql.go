package sql

import (
	"context"
	"io/ioutil"
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	// MySQL drivers
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	// Postgres driver
	_ "github.com/lib/pq"

	// Oracle
	_ "github.com/sijms/go-ora"

	"github.com/ovh/venom"
)

// Name of the executor.
const Name = "sql"

// New returns a new executor that can execute SQL queries
func New() venom.Executor {
	return &Executor{}
}

// Executor is a venom executor can execute SQL queries
type Executor struct {
	File     string   `json:"file,omitempty" yaml:"file,omitempty"`
	Commands []string `json:"commands,omitempty" yaml:"commands,omitempty"`
	Driver   string   `json:"driver" yaml:"driver"`
	DSN      string   `json:"dsn" yaml:"dsn"`
}

// Rows represents an array of Row
type Rows []Row

// Row represents a row return by a SQL query.
type Row map[string]interface{}

// QueryResult represents a rows return by a SQL query execution.
type QueryResult struct {
	Rows Rows `json:"rows,omitempty" yaml:"rows,omitempty"`
}

// Result represents a step result.
type Result struct {
	Queries []QueryResult `json:"queries,omitempty" yaml:"queries,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	venom.Debug(ctx, "connecting to database %s, %s\n", e.Driver, e.DSN)
	db, err := sqlx.Connect(e.Driver, e.DSN)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to database")
	}
	defer db.Close()

	results := []QueryResult{}
	// Execute commands on database
	// if the argument is specified.
	if len(e.Commands) != 0 {
		for i, s := range e.Commands {
			venom.Debug(ctx, "Executing command number %d\n", i)
			rows, err := db.Queryx(s)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to exec command number %d", i)
			}
			r, err := handleRows(rows)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse SQL rows for command number %d", i)
			}
			results = append(results, QueryResult{Rows: r})
		}
	} else if e.File != "" {
		workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")
		file := path.Join(workdir, e.File)
		venom.Debug(ctx, "loading SQL file from %s\n", file)
		sbytes, errs := ioutil.ReadFile(file)
		if errs != nil {
			return nil, errs
		}
		rows, err := db.Queryx(string(sbytes))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to exec SQL file %q", file)
		}
		r, err := handleRows(rows)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse SQL rows for SQL file %q", file)
		}
		results = append(results, QueryResult{Rows: r})
	}
	r := Result{Queries: results}
	return r, nil
}

// ZeroValueResult return an empty implementation of this executor result
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions return the default assertions of the executor.
func (e Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{}}
}

// handleRows iter on each SQL rows result sets and serialize it into a []Row.
func handleRows(rows *sqlx.Rows) ([]Row, error) {
	defer rows.Close()
	res := []Row{}
	for rows.Next() {
		row := make(Row)
		if err := rows.MapScan(row); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	return res, nil
}
