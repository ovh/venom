package spanner

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"cloud.google.com/go/spanner"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/ovh/venom"
)

// Name of the executor.
const Name = "spanner"

// New returns a new executor that can execute SQL queries and DML statements on Spanner.
func New() venom.Executor {
	return &Executor{}
}

// Executor is a venom executor that can execute SQL and DML statements on Google Cloud Spanner.
type Executor struct {
	Project         string   `json:"project,omitempty" yaml:"project,omitempty"`
	Instance        string   `json:"instance,omitempty" yaml:"instance,omitempty"`
	Database        string   `json:"database,omitempty" yaml:"database,omitempty"`
	CredentialsFile string   `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty" mapstructure:"credentials_file"`
	Commands        []string `json:"commands,omitempty" yaml:"commands,omitempty"`
	File            string   `json:"file,omitempty" yaml:"file,omitempty"`
}

// Row represents a row returned by a query.
type Row map[string]interface{}

// QueryResult represents the output of one SQL statement.
type QueryResult struct {
	StatementType string `json:"statement_type,omitempty" yaml:"statement_type,omitempty"`
	Rows          []Row  `json:"rows,omitempty" yaml:"rows,omitempty"`
	RowCount      int64  `json:"row_count,omitempty" yaml:"row_count,omitempty"`
}

// Result represents a step result.
type Result struct {
	Queries []QueryResult `json:"queries,omitempty" yaml:"queries,omitempty"`
}

// Run implements the venom.Executor interface for Executor.
func (e Executor) Run(ctx context.Context, step venom.TestStep) (interface{}, error) {
	if err := mapstructure.Decode(step, &e); err != nil {
		return nil, err
	}

	if err := e.validate(); err != nil {
		return nil, err
	}

	commands, err := e.commandsFromStep(ctx)
	if err != nil {
		return nil, err
	}

	client, err := e.newClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	results := make([]QueryResult, 0, len(commands))
	for i, cmd := range commands {
		stmt := strings.TrimSpace(cmd)
		if stmt == "" {
			continue
		}

		venom.Debug(ctx, "spanner: executing statement number %d\n", i)
		result, execErr := executeStatement(ctx, client, stmt)
		if execErr != nil {
			return nil, errors.Wrapf(execErr, "failed to execute statement number %d", i)
		}
		results = append(results, result)
	}

	return Result{Queries: results}, nil
}

// ZeroValueResult returns an empty implementation of this executor result.
func (Executor) ZeroValueResult() interface{} {
	return Result{}
}

// GetDefaultAssertions returns the default assertions of the executor.
func (Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []venom.Assertion{}}
}

func (e Executor) validate() error {
	if e.Project == "" {
		return fmt.Errorf("missing project")
	}
	if e.Instance == "" {
		return fmt.Errorf("missing instance")
	}
	if e.Database == "" {
		return fmt.Errorf("missing database")
	}
	if len(e.Commands) == 0 && e.File == "" {
		return fmt.Errorf("commands or file is required")
	}
	return nil
}

func (e Executor) commandsFromStep(ctx context.Context) ([]string, error) {
	if len(e.Commands) > 0 {
		return e.Commands, nil
	}

	workdir := venom.StringVarFromCtx(ctx, "venom.testsuite.workdir")
	file := path.Join(workdir, e.File)
	venom.Debug(ctx, "spanner: loading SQL file from %s\n", file)

	b, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read SQL file %q", file)
	}
	stmt := strings.TrimSpace(string(b))
	if stmt == "" {
		return nil, fmt.Errorf("SQL file %q is empty", file)
	}
	return []string{stmt}, nil
}

func (e Executor) newClient(ctx context.Context) (*spanner.Client, error) {
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", e.Project, e.Instance, e.Database)

	if e.CredentialsFile == "" {
		venom.Debug(ctx, "spanner: connecting to %s with ADC\n", db)
		client, err := spanner.NewClient(ctx, db)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to connect to spanner")
		}
		return client, nil
	}

	currentCreds, hadCreds := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
	if err := os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", e.CredentialsFile); err != nil {
		return nil, errors.Wrap(err, "failed to set GOOGLE_APPLICATION_CREDENTIALS")
	}

	client, err := spanner.NewClient(ctx, db)

	// Restore process state right away to avoid side effects for subsequent steps.
	if hadCreds {
		_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", currentCreds)
	} else {
		_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to spanner with credentials_file")
	}
	return client, nil
}

func executeStatement(ctx context.Context, client *spanner.Client, query string) (QueryResult, error) {
	if isDML(query) {
		rowCount, err := executeDML(ctx, client, query)
		if err != nil {
			return QueryResult{}, err
		}
		return QueryResult{
			StatementType: "dml",
			RowCount:      rowCount,
		}, nil
	}

	rows, err := executeQuery(ctx, client, query)
	if err != nil {
		return QueryResult{}, err
	}
	return QueryResult{
		StatementType: "query",
		Rows:          rows,
	}, nil
}

func isDML(query string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(query))
	switch {
	case strings.HasPrefix(normalized, "INSERT "),
		strings.HasPrefix(normalized, "UPDATE "),
		strings.HasPrefix(normalized, "DELETE "):
		return true
	default:
		return false
	}
}

func executeDML(ctx context.Context, client *spanner.Client, query string) (int64, error) {
	stmt := spanner.Statement{SQL: query}
	var rowCount int64
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		n, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		rowCount = n
		return nil
	})
	if err != nil {
		return 0, err
	}
	return rowCount, nil
}

func executeQuery(ctx context.Context, client *spanner.Client, query string) ([]Row, error) {
	stmt := spanner.Statement{SQL: query}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	rows := []Row{}
	err := iter.Do(func(r *spanner.Row) error {
		row, err := spannerRowToMap(r)
		if err != nil {
			return err
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func spannerRowToMap(row *spanner.Row) (Row, error) {
	columns := row.ColumnNames()
	m := make(Row, len(columns))
	for i, name := range columns {
		v, err := decodeColumnValue(row, i)
		if err != nil {
			return nil, err
		}
		m[name] = v
	}
	return m, nil
}

func decodeColumnValue(row *spanner.Row, i int) (interface{}, error) {
	t := row.ColumnType(i)
	if t == nil {
		// Should not happen for real results, but be defensive.
		var g spanner.GenericColumnValue
		if err := row.Column(i, &g); err != nil {
			return nil, err
		}
		return g, nil
	}

	switch t.Code {
	case sppb.TypeCode_STRING:
		var v spanner.NullString
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.StringVal, nil

	case sppb.TypeCode_INT64:
		var v spanner.NullInt64
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Int64, nil

	case sppb.TypeCode_BOOL:
		var v spanner.NullBool
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Bool, nil

	case sppb.TypeCode_FLOAT64:
		var v spanner.NullFloat64
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Float64, nil

	case sppb.TypeCode_FLOAT32:
		var v spanner.NullFloat32
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Float32, nil

	case sppb.TypeCode_BYTES:
		var v []byte
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		// If column is NULL, v is nil.
		return v, nil

	case sppb.TypeCode_TIMESTAMP:
		var v spanner.NullTime
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Time, nil

	case sppb.TypeCode_DATE:
		var v spanner.NullDate
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Date, nil

	case sppb.TypeCode_JSON:
		var v spanner.NullJSON
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.Value, nil

	case sppb.TypeCode_NUMERIC:
		var v spanner.NullNumeric
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		// Keep it assertion-friendly.
		return v.Numeric.String(), nil

	case sppb.TypeCode_UUID:
		var v spanner.NullUUID
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return v.UUID.String(), nil

	case sppb.TypeCode_STRUCT:
		var v spanner.NullRow
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		if !v.Valid {
			return nil, nil
		}
		return spannerRowToMap(&v.Row)

	case sppb.TypeCode_ARRAY:
		return decodeArrayColumnValue(row, i, t.ArrayElementType)

	default:
		// Fallback to the encoded form for uncommon types (e.g. PG types).
		var g spanner.GenericColumnValue
		if err := row.Column(i, &g); err != nil {
			return nil, err
		}
		return g, nil
	}
}

func decodeArrayColumnValue(row *spanner.Row, i int, elem *sppb.Type) (interface{}, error) {
	if elem == nil {
		var g spanner.GenericColumnValue
		if err := row.Column(i, &g); err != nil {
			return nil, err
		}
		return g, nil
	}

	switch elem.Code {
	case sppb.TypeCode_STRING:
		var v []spanner.NullString
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.StringVal)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_INT64:
		var v []spanner.NullInt64
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Int64)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_BOOL:
		var v []spanner.NullBool
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Bool)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_FLOAT64:
		var v []spanner.NullFloat64
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Float64)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_FLOAT32:
		var v []spanner.NullFloat32
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Float32)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_TIMESTAMP:
		var v []spanner.NullTime
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Time)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_DATE:
		var v []spanner.NullDate
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Date)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_JSON:
		var v []spanner.NullJSON
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Value)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_NUMERIC:
		var v []spanner.NullNumeric
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.Numeric.String())
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_UUID:
		var v []spanner.NullUUID
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				out = append(out, e.UUID.String())
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_STRUCT:
		var v []spanner.NullRow
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		out := make([]interface{}, 0, len(v))
		for _, e := range v {
			if e.Valid {
				m, err := spannerRowToMap(&e.Row)
				if err != nil {
					return nil, err
				}
				out = append(out, m)
			} else {
				out = append(out, nil)
			}
		}
		return out, nil

	case sppb.TypeCode_BYTES:
		// Array of BYTES.
		var v [][]byte
		if err := row.Column(i, &v); err != nil {
			return nil, err
		}
		return v, nil

	default:
		var g spanner.GenericColumnValue
		if err := row.Column(i, &g); err != nil {
			return nil, err
		}
		return g, nil
	}
}
