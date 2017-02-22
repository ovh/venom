# Venom - Executor Web

Navigate in a web application

Use case: You have a web application and you want to check some behaviours ?
Venom allows you to navigate into it and execute actions.

## Input

```yaml
name: TestSuite Web
testcases:
- name: TestCase Get URL and check title
  context:
    type: web
  steps:
  - type: web
    action: navigate
    url: http://www.google.fr
  - type: web
    action: title
    assertions:
      - result.title ShouldEqual Google

```

## Output

* result.timeseconds
* result.timehuman
* result.title
* result.error

## Action
* navigate: navigate to url
* title: get title value

more actions are coming... 