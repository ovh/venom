# Venom - Executor Spanner

Step to execute SQL statements on Google Cloud Spanner.

Supports:
* Query statements (for example `SELECT`, `WITH`)
* DML statements (for example `INSERT`, `UPDATE`, `DELETE`)

## Input

In your yaml file, declare your step like this:

```yaml
  - project mandatory
  - instance mandatory
  - database mandatory
  - credentials_file optional
  - commands optional
  - file optional
```

- `commands` is a list of SQL statements.
- `file` is used as fallback if `commands` is empty.
- `credentials_file` is optional. If not set, the executor uses Application Default Credentials (ADC).

## Output

Each statement result is returned in `result.queries`:

- Query statement:
  - `statement_type: query`
  - `rows: []`
- DML statement:
  - `statement_type: dml`
  - `row_count: <number>`

## Example usage

```yaml
name: Spanner sample
testcases:
  - name: Query and DML
    steps:
      - type: spanner
        project: my-project
        instance: my-instance
        database: my-database
        commands:
          - "SELECT SingerId, FirstName FROM Singers LIMIT 1"
          - "UPDATE Singers SET FirstName = 'Alice' WHERE SingerId = 1"
        assertions:
          - result.queries.__Len__ ShouldEqual 2
          - result.queries.queries0.statement_type ShouldEqual query
          - result.queries.queries1.statement_type ShouldEqual dml
```

Example using a credentials file:

```yaml
name: Spanner with SA key
testcases:
  - name: Query with credential file
    steps:
      - type: spanner
        project: my-project
        instance: my-instance
        database: my-database
        credentials_file: /path/to/service-account.json
        commands:
          - "SELECT 1"
```
