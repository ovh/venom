## Variable helpers examples

default value with empty default:
- `aa:{{.myvar.foo | default ""}}end`
- with no variable defined
- will return `aa:end`

default value:
- `aa:{{.myvar.foo | default "bar" }}:end`
- with no variable defined
- will return `aa:bar:end`

default value with variables (easy):
- `{{.myvar.foo | default .val }}`
- with variable `val=biz`
- will return `biz`

default value with variables (not so easy):
- `{{.myvar.foo | default .myvar.bar }}`
- with variable `myvar.bar=biz`
- will return `biz`

default value with variables (so hard):
- `{{.myvar.foo | default .myvar.bar .myvar.biz }}`
- with variable `myvar.biz=biz`
- will return `biz`

default value with variables (with pipeline):
- `{{.myvar.foo | default .myvar.bar | default .myvar.biz | upper }}`
- with variable `myvar.biz=biz`
- will return `BIZ`

default value with knowned var:
- `aa:{{.myvar.foo | default "bar"}}end`
- with variable `myvar.foo=value`
- will return `aa:valueend`

default empty value with knowned var:
- `aa:{{.myvar.foo | default ""}}end`
- with variable `myvar.foo=value`
- will return `aa:valueend`

unknown function:
- `echo '{{"conf"|uvault}}'`
- with no variable defined
- will return `echo '{{"conf"|uvault}}'`

simple:
- `a {{.myvar.value}}`
- with variable `myvar.value=value`
- will return `a value`

only unknown:
- `a value unknown {{.myvar.foo}}`
- with variable `myvar.value=value`
- will return `a value unknown {{.myvar.foo}}`

simple with unknown:
- `a {{.myvar.value}} and another value unknown {{.myvar.foo}}`
- with variable `myvar.value=value`
- will return `a value and another value unknown {{.myvar.foo}}`

upper:
- `a {{.myvar.value | upper}} and another value unknown {{.myvar.foo}}`
- with variable `myvar.value=value`
- will return `a VALUE and another value unknown {{.myvar.foo}}`

title and filter on unknow:
- `a {{.myvar.value | title }} and another value unknown {{.myvar.foo | lower}}`
- with variable `myvar.value=value`
- will return `a Value and another value unknown {{.myvar.foo | lower}}`

many:
- `{{.myvar.bar}} a {{.myvar.valuea | upper }}, a {{.myvar.valueb | title}}.{{.myvar.valuec}}-{{.myvar.foo}}`
- with variables 
    - `myvar.valuea=valuea`
    - `myvar.valueb=valueb`
    - `myvar.valuec=valuec`
- will return `{{.myvar.bar}} a VALUEA, a Valueb.valuec-{{.myvar.foo}}`

empty string:
- `a {{.myvar.myKey}} and another key with empty value *{{.myvar.myKeyAnother}}*`
- with variables
    - `myvar.myKey=valueKey`
    - `myvar.myKeyAnother=""`
- will return `a valueKey and another key with empty value **`

two keys with same first characters:
- `a {{.myvar.myKey}} and another key value {{.myvar.myKeyAnother}}`
- with variables 
  - `myvar.myKey=valueKey`
  - `myvar.myKeyAnother=valueKeyAnother`
- will return `a valueKey and another key value valueKeyAnother`

key with - and a unknown key:
- `a {{.myvar.my-key}}.{{.myvar.foo-key}} and another key value {{.myvar.my-key}}`
- with variable `myvar.my-key=value-key`
- will return `a value-key.{{.myvar.foo-key}} and another key value value-key`

key with - and a empty key:
- `a {{.myvar.my-key}}.{{.myvar.foo-key}}.and another key value {{.myvar.my-key}}`
- with variables 
    - `myvar.my-key=value-key`
    - `myvar.foo-key=""`
- will return `a value-key..and another key value value-key`

escape func:
- `a {{.m.foo}} here, {{.m.title | title}}, {{.m.upper | upper}}, {{.m.lower | lower}}, {{.m.escape | escape}}`
- with variables 
  - `m.foo=valbar`
  - `m.title=mytitle-bis`
  - `m.upper=toupper`
  - `m.lower=TOLOWER`
  - `m.escape=a/b.c_d`
- will return `a valbar here, Mytitle-Bis, TOUPPER, tolower, a-b-c-d`

substring:
- `name: hello-{{ .name | substr 0 5 }}`
- with variable `name=github`
- will return `name: hello-githu`

trunc:
- `test_{{.m.workflow}}_{{.git.hash | trunc 8 }}`
- with variables 
  - `m.workflow=myWorkflow`
  - `git.hash=863ddke13bfef8043960b19cec790f8b9f5435ab`
  - `git.hash.before=863ddke13bfef8043960b19cec790f8b9f5435ab`
- will return `test_myWorkflow_863ddke1`

add:
- `my value {{.myvar.value | add 3}} {{ add 2 2 }}`
- with variable `myvar.value=1`
- will return `my value 4 4`

dirname:
- `{{.path | dirname}}`
- with variable `path=/a/b/c`
- will return `/a/b`

basename:
- `{{.path | basename}}`
- with variable `path=/ab/c`
- will return `c`

urlencode word:
- `{{.query | urlencode}}`
- with variable `query=Trollhättan`
- will return `Trollh%C3%A4ttan`

urlencode query:
- `{{.query | urlencode}}`
- with variable `query=zone:eq=Somewhere over the rainbow&name:like=%mydomain.localhost.local`
- will return `zone%3Aeq%3DSomewhere+over+the+rainbow%26name%3Alike%3D%25mydomain.localhost.local`

urlencode nothing to do:
- `{{.query | urlencode}}`
- with variable `query=patrick`
- will return `patrick`

ternary truthy:
- `{{.assert | ternary .foo .bar}}`
- with variables
  - `assert=true`
  - `bar=bar`
  - `foo=foo`
- will return `foo`

ternary truthy integer:
- `{{ \"1\" | ternary .foo .bar}}`
- with variables 
  - `bar=bar`
  - `foo=foo`
- will return `foo`

ternary falsy:
- `{{.assert | ternary .foo .bar}}`
- with variables 
  - `assert=false`
  - `bar=bar`
  - `foo=foo`
- will return `bar`

ternary undef assert:
- `{{.assert | ternary .foo .bar}}`
- with variables 
  - `bar=bar`
  - `foo=foo`
- will return `bar`