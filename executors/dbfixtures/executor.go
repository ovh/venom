package dbfixtures

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"path"

	fixtures "github.com/go-testfixtures/testfixtures/v3"
	"github.com/mitchellh/mapstructure"
	migrate "github.com/rubenv/sql-migrate"

	// SQL drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/ovh/venom"
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
	Files              []string `json:"files" yaml:"files"`
	Folder             string   `json:"folder" yaml:"folder"`
	Database           string   `json:"database" yaml:"database"`
	DSN                string   `json:"dsn" yaml:"dsn"`
	Schemas            []string `json:"schemas" yaml:"schemas"`
	Migrations         string   `json:"migrations" yaml:"migrations"`
	MigrationsTable    string   `json:"migrationsTable" yaml:"migrationsTable"`
	SkipResetSequences bool     `json:"skipResetSequences" yaml:"skipResetSequences"`
}

// Result represents a step result.
type Result struct {
	Executor Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(ctx context.Context, testCaseContext venom.TestCaseContext, step venom.TestStep, workdir string) (interface{}, error) {
	// Transform step to Executor instance.
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}
	// Connect to the database and ping it.
	venom.Debug(ctx, "connecting to database %s, %s\n", e.Database, e.DSN)

	db, err := sql.Open(e.Database, e.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	// Load and import the schemas in the database
	// if the argument is specified.
	if len(e.Schemas) != 0 {
		for _, s := range e.Schemas {
			venom.Debug(ctx, "loading schema from file %s\n", s)
			s = path.Join(workdir, s)
			sbytes, errs := ioutil.ReadFile(s)
			if errs != nil {
				return nil, errs
			}
			if _, err = db.Exec(string(sbytes)); err != nil {
				return nil, fmt.Errorf("failed to exec schema from file %s : %v", s, err)
			}
		}
	} else if e.Migrations != "" {
		venom.Debug(ctx, "loading migrations from folder %s\n", e.Migrations)

		if e.MigrationsTable != "" {
			migrate.SetTable(e.MigrationsTable)
		}

		dir := path.Join(workdir, e.Migrations)
		migrations := &migrate.FileMigrationSource{
			Dir: dir,
		}
		n, errMigrate := migrate.Exec(db, e.Database, migrations, migrate.Up)
		if errMigrate != nil {
			return nil, fmt.Errorf("failed to apply up migrations: %s", errMigrate)
		}
		venom.Debug(ctx, "applied %d migrations\n", n)
	}

	// Load fixtures in the databases.
	if err = loadFixtures(ctx, db, e.Files, e.Folder, getDialect(e.Database, e.SkipResetSequences), workdir); err != nil {
		return nil, err
	}
	r := Result{Executor: e}

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

// loadFixtures loads the fixtures in the database.
// It gives priority to the fixtures files found in folder,
// and switch to the list of files if no folder was specified.
func loadFixtures(ctx context.Context, db *sql.DB, files []string, folder string, dialect func(*fixtures.Loader) error, workdir string) error {
	if folder != "" {
		venom.Debug(ctx, "loading fixtures from folder %s\n", path.Join(workdir, folder))
		loader, err := fixtures.New(
			// By default the package refuse to load if the database
			// does not contains "test" to avoid wiping a production db.
			fixtures.DangerousSkipTestDatabaseCheck(),
			fixtures.Database(db),
			fixtures.Directory(path.Join(workdir, folder)),
			dialect)

		if err != nil {
			return fmt.Errorf("failed to create folder loader: %v", err)
		}
		if err = loader.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from folder %s: %v", path.Join(workdir, folder), err)
		}
		return nil
	}
	if len(files) != 0 {
		venom.Debug(ctx, "loading fixtures from files: %v\n", files)
		for i := range files {
			files[i] = path.Join(workdir, files[i])
		}
		loader, err := fixtures.New(
			// By default the package refuse to load if the database
			// does not contains "test" to avoid wiping a production db.
			fixtures.DangerousSkipTestDatabaseCheck(),
			fixtures.Database(db),
			fixtures.Files(files...),
			dialect)

		if err != nil {
			return fmt.Errorf("failed to create files loader: %v", err)
		}
		if err = loader.Load(); err != nil {
			return fmt.Errorf("failed to load fixtures from files: %v", err)
		}
		return nil
	}
	venom.Debug(ctx, "neither files or folder parameter was used\n")

	return nil
}

func getDialect(name string, skipResetSequences bool) func(*fixtures.Loader) error {
	switch name {
	case "postgres":
		return func(l *fixtures.Loader) error {
			if err := fixtures.Dialect("postgresql")(l); err != nil {
				return err
			}
			if skipResetSequences {
				if err := fixtures.SkipResetSequences()(l); err != nil {
					return err
				}
			}
			return nil
		}
	case "mysql":
		return fixtures.Dialect("mysql")
	}
	return nil
}
