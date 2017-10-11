# Venom - Executor Database Fixtures

Step to load fixtures into **MySQL** and **PostgreSQL** databases.

It use the package `testfixtures.v2` under the hood: https://github.com/go-testfixtures/testfixtures
Please read its documentation for further details about the parameters of this executor, especially `folder` and `files`, and how you should write the fixtures.

## Input
In your yaml file, you declare tour step like this

```yaml
  - database mandatory [mysql/postgres]
  - dsn mandatory
  - schema optional
  - files optional
  - folder optional
 ```

- `schema` is the path to a `.sql` file that contains the schema of your database. If it present, the content will be executed before loading the fixtures.
- If `folder` is specified, the executor won't use the `files` parameter.

Example usage (_mysql_):
```yaml

name: Title of TestSuite
testcases:

  - name: Load database fixtures
    steps:
      - type: dbfixtures
        database: mysql
        dsn: user:password@(localhost:3306)/venom?multiStatements=true
        schema: schemas/mysql.sql
        folder: fixtures

```

*note: in the example above, the query param `multiStatements=true` is mandatory if we want to be able to load the schema.*

## SQL drivers

This executor uses the following SQL drivers:

- _MySQL_: https://github.com/go-sql-driver/mysql
- _PostgreSQL_: https://github.com/lib/pq
