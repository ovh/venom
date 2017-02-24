# Venom - Executor Web

Navigate in a web application

Use case: You have a web application and you want to check some behaviours ?
Venom allows you to navigate into it and execute actions.

## Input

* Action (https://github.com/runabove/venom/tree/master/executors/web/types.go)
* Screenshot

```yaml
name: TestSuite Web
testcases:
- name: TestCase Google search
  context:
    type: web
    width: 1920
    height: 1080
  steps:
  - type: web
    action:
      navigate: https://www.google.fr
    assertions:
    - result.title ShouldEqual Google
    - result.url ShouldEqual https://www.google.fr
  - type: web
    action:
      find: input[name="q"]
    assertions:
     - result.find ShouldEqual 1
  - type: web
    action:
      fill:
        find: input[name="q"]
        text: "venom runabove"
  - type: web
    action:
      click: input[value="Recherche Google"]
    screenshot: googlesearch.jpg

```

## Output

* result.url
* result.timeseconds
* result.timehuman
* result.title
* result.find
* result.html