package dbfixtures

import (
	"database/sql"
	"fmt"
	"io/ioutil"

	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	fixtures "gopkg.in/testfixtures.v2"

	// SQL drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Name of the executor.
const Name = "dbfixtures"

// New returns a new executor that can load
// database fixtures.
func New() venom.Executor {
	return &Executor{}
}

// Executor is a venom executor that can load
// fixtures in many databases, using YAML schemas.
type Executor struct {
	Files    []string `json:"files" yaml:"files"`
	Folder   string   `json:"folder" yaml:"folder"`
	Database string   `json:"database" yaml:"database"`
	DSN      string   `json:"dsn" yaml:"dsn"`
	Schema   string   `json:"schema" yaml:"schema"`
}

// Result represents a step result.
type Result struct {
	Executor Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {
	// Transform step to Executor instance.
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	l.Debugf("connecting to database %s, %s\n", t.Database, t.DSN)

	db, err := sql.Open(t.Database, t.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	// Load and import the schema in the database
	// if the argument is specified.
	if t.Schema != "" {
		l.Debugf("loading schema from file %s\n", t.Schema)

		schema, errs := ioutil.ReadFile(t.Schema)
		if errs != nil {
			return nil, errs
		}
		_, errs = db.Exec(string(schema))
		if errs != nil {
			return nil, fmt.Errorf("failed to exec schema: %v", errs)
		}
	}
	// Load fixtures in the databases.
	// Bu default the package refuse to load if the database
	// does not contains test to avoid wiping a production db.
	fixtures.SkipDatabaseNameCheck(true)
	err = loadFixtures(db, t.Files, t.Folder, databaseHelper(t.Database), l)
	if err != nil {
		return nil, err
	}
	r := Result{Executor: t}

	return dump.ToMap(r)
}

// GetDefaultAssertions return the default assertions of the executor.
func (e Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{}}
}

// loadFixtures loads the fixtures in the database.
// It gives priority to the fixtures files found in folder,
// and switch to the list of files if no folder was specified.
func loadFixtures(db *sql.DB, files []string, folder string, helper fixtures.Helper, l venom.Logger) error {
	if folder != "" {
		l.Debugf("loading fixtures from folder %s\n", folder)

		c, err := fixtures.NewFolder(db, helper, folder)
		if err != nil {
			return fmt.Errorf("failed to create folder context: %v", err)
		}
		if err = c.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from folder %s: %v", folder, err)
		}
		return nil
	}
	if len(files) != 0 {
		l.Debugf("loading fixtures from files: %v\n", files)

		c, err := fixtures.NewFiles(db, helper, files...)
		if err != nil {
			return fmt.Errorf("failed to create files context: %v", err)
		}
		if err = c.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from files: %v", err)
		}
		return nil
	}
	l.Debugf("neither files or folder parameter was used\n")

	return nil
}

func databaseHelper(name string) fixtures.Helper {
	switch name {
	case "postgres":
		return &fixtures.PostgreSQL{}
	case "mysql":
		return &fixtures.MySQL{}
	}
	return nil
}
