package sql

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/mitchellh/mapstructure"

	// MySQL drivers
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	// Postgres driver
	_ "github.com/lib/pq"

	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
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
	Commands []string `json:"commands,omitempy" yaml:"commands,omitempty"`
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
	Executor Executor      `json:"executor,omitempty" yaml:"executor,omitempty"`
	Queries  []QueryResult `json:"queries,omitempty" yaml:"queries,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep, workdir string) (venom.ExecutorResult, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	l.Debugf("connecting to database %s, %s\n", e.Driver, e.DSN)
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
			l.Debugf("Executing command number %d\n", i)
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
		l.Debugf("loading SQL file from folder %s\n", e.File)
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
	r := Result{Executor: e, Queries: results}
	return executors.Dump(r)
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
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
