# Venom - Executor SQL

This executor is a plugin executor. You need to recompile venom with plugin support to use it.

```bash
$ git clone https://github.com/ovh/venom.git
$ cd venom
$ make build
$ make plugins
$ # venom binary is generated into dist directory
$ # plugin binary is generated into dist/lib directory
$ cd dist
$ ./venom run ... 
```

To compile this executor, you need an ODBC driver. Please read https://github.com/alexbrainman/odbc/wiki

Example on osx with driver installed with homebrew:
```
$ CGO_CFLAGS="-I$HOME/homebrew/Cellar/unixodbc/2.3.9/include" CGO_LDFLAGS="-L$HOME/homebrew/lib" go build
```

Step to execute SQL queries into databases:

* **ODBC**

It use the package `sqlx` under the hood: https://github.com/jmoiron/sqlx to retreive rows as a list of map[string]interface{}

## Input

In your yaml file, you declare tour step like this

```yaml
  - dsn mandatory
  - commands optional
  - file optional
 ```

- `commands` is a list of SQL queries.
- `file` parameter is only used as a fallback if `commands` is not used.

Example usage (_mysql_, _oracle_, _SQLServer_):

```yaml
name: Title of TestSuite
testcases:

  - name: Query database
    steps:
      - type: odbc
        dsn: user:password@(localhost:3306)/venom
        commands:
          - "SELECT * FROM employee;"
          - "SELECT * FROM person;"
        assertions:
          - result.queries.__len__ ShouldEqual 2
          - result.queries.queries0.rows.rows0.name ShouldEqual Jack
          - result.queries.queries1.rows.rows0.age ShouldEqual 21
```

Example with a query file:

```yaml
name: Title of TestSuite
testcases:

  - name: Query database
    steps:
      - type: odbc
        database: thedatabase
        dsn: user:password@(localhost:3306)/venom
        file: ./test.sql
        assertions:
          - result.queries.__len__ ShouldEqual 1
```

*note: in the example above, the results of each command is stored in the results array

## SQL drivers

This executor uses the following SQL drivers:

- _ODBC_: https://github.com/alexbrainman/odbc
