# Venom - Executor Tavern

This executor is a [Tavern](https://tavern.readthedocs.io/en/latest/) like executor to test HTTP responses against expected one. For instance, to call a REST API that returns a user, you might write:

```json
name: User
vars:
  URL: "http://127.0.0.1:8082/v1"

testcases:

- name: Get user Adeline
  steps:
  - type: tavern
    request:
      url: "{{.URL}}/user/1"
      method: GET
      headers:
        Authorization: Bearer VENOM
    response:
      statusCode: 200
      headers:
        Content-Type: "application/json; charset=utf-8"
      json:
        ID:        1
        FirstName: "Adeline"
        LastName:  "Durand"
        Login:     "adeline"
        Role:
          ID: 1
          Name: "Reader"
          Permissions:
          - ID: 1
            Name: "Read"
        Organizations:
        - ID:         1
          Name:       "Intercloud"
          Address:    "88 bd Sébastopol"
          PostalCode: "75003"
          City:       "Paris"
      jsonExcludes:
      - "**/CreationDate"
```

This will perform the call described in `request` field and test response as described in `response` field excluding *CreationDate* fields at any level in JSON.

This executor is based on `http` executor and calls utility functions defined in *httputil.go* file.

## Request

Possible fields are:

- **url**: the URL to call
- **method**: HTTP method for request
- **headers**: a map of headers for request
- **body**: request body as text
- **file**: request body as file
- **json**: request body as JSON in a map

For *multipartform*, *basicauth*, *ignoreverifyssl*, *proxy*, *resolve*, *nofollowredirect*, *unixsock*, *tls*, *skipbody* and *skipheaders*, please see [HTTP executor documentation](https://github.com/ovh/venom/tree/master/executors/http).

## Response

Possible fields are:

- **StatusCode**: expected status code
- **Headers**: expected headers as a map (only defined headers are tested)
- **Body**: expected text body
- **Json**: expected JSON body as a structure
- **JsonExcludes**: a list of paths to excludes from test in JSON response

Fields that are not set are not checked. Only defined headers are checked, thus Tavern executor won't complain about an additional header.

You can exclude paths from JSON during test. For instance, to exclude field `CreationDate` during response comparison, you might add in `response` field:

```yaml
jsonExcludes:
- "CreationDate"
```

To exclude all `CreationDate` fields at second level from comparison, you can write:

```yaml
jsonExcludes:
- "*/CreationDate"
```

To exclude at any level, you would write:

```yaml
jsonExcludes:
- "**/CreationDate"
```

To exclude it in the first entry:

```yaml
jsonExcludes:
- "0/CreationDate"
```

Thus:

- **text** matches given entry
- **\*** matches any entry
- **\*\*** matches any entries in successive levels

## Default Assertions

One default assertion is made in this executor:

```
result AssertResponse
```

`result` is made of fields `expected`, `actual` and `timeseconds`:

- **expected** is filled with expected response as defined in `response` field in test.
- **actual** is filled with actual server response.
- **timeseconds** is the time the request took to perform in seconds.

If you want to perform assertions in addition to response one, you must write these in addition of default one. For instance, if you want to add an assertion on the execution time, you would write:

```json
assertions:
- result AssertResponse
- result.timeseconds ShouldBeLessThan 0.1
```

## Note About Types

If you expected and actual JSON are of different types (*list* and *map* for instance), you wil have an error message as the diff engine will be unable to generate a diff between these two type. In this case, you will have an error as follows in your test result:

```
generating JSON diff: types do not match (cause count 0)
```

## YAML References

If you have fields to excludes in a lot of tests, you can define them in lists at the top of your test and reference them in your test definitions, as follows:

```yaml
name: Test
excludes: &excludes ["**/CreatedAt", "**/UpdatedAt", "**/DeletedAt"]

testcases:

- name: Get user
  steps:
  - type: tavern
    request:
      url: "{{.URL}}/user/1"
      method: GET
    response:
      statusCode: 200
      json:
        ID:        1
        FirstName: "Adeline"
        LastName:  "Durand"
      jsonExcludes: &excludes
...

*Enjoy!*
