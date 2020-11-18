package main

import (
	"C"
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/mitchellh/mapstructure"

	// MySQL drivers
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	// Postgres driver
	_ "github.com/lib/pq"

	// Oracle
	_ "github.com/sijms/go-ora"

	// ODBC
	_ "github.com/alexbrainman/odbc"

	"github.com/ovh/venom"
)

// Name of the executor.
const Name = "sql"

// Plugin var is mandatory, it's used by venom to register the executor
var Plugin = Executor{}

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
func (e Executor) Run(ctx context.Context, step venom.TestStep, workdir string) (interface{}, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	venom.Debug(ctx, "connecting to database %s, %s\n", e.Driver, e.DSN)
	db, err := sqlx.Connect(e.Driver, e.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
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
				return nil, fmt.Errorf("failed to exec command number %d : %v", i, err)
			}
			r, err := handleRows(rows)
			if err != nil {
				return nil, fmt.Errorf("failed to parse SQL rows for command number %d : %v", i, err)
			}
			results = append(results, QueryResult{Rows: r})
		}
	} else if e.File != "" {
		venom.Debug(ctx, "loading SQL file from folder %s\n", e.File)
		file := path.Join(workdir, e.File)
		sbytes, errs := ioutil.ReadFile(file)
		if errs != nil {
			return nil, errs
		}
		rows, err := db.Queryx(string(sbytes))
		if err != nil {
			return nil, fmt.Errorf("failed to exec SQL file %s : %v", file, err)
		}
		r, err := handleRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SQL rows for SQL file %s : %v", file, err)
		}
		results = append(results, QueryResult{Rows: r})
	}
	r := Result{Queries: results}
	return r, nil
}

// ZeroValueResult return an empty implemtation of this executor result
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
