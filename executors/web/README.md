# Venom - Executor Web

Navigate in a web application

Use case: You have a web application and you want to check some behaviours?
Venom allows you to navigate into it and execute actions.

## Input

* Action (https://github.com/ovh/venom/tree/master/executors/web/types.go)
* Format

Parameters `debug` (default: false) and `timeout` (default: 180 seconds) are optional.

```yaml
name: TestSuite Web
testcases:
- name: TestCase Google search
  context:
    type: web
    width: 1920
    height: 1080
    driver: phantomjs
    timeout: 60
    debug: true
  steps:
  - action:
      navigate:
        url: https://www.google.fr
    assertions:
    - result.title ShouldEqual Google
    - result.url ShouldEqual https://www.google.fr
  - action:
      find: input[name="q"]
    assertions:
     - result.find ShouldEqual 1
  - action:
      fill:
      - find: input[name="q"]
        text: "venom ovh"
  - action:
      click:
        find: input[value="Recherche Google"]
        wait: 1
    screenshot: googlesearch.jpg

```

Select statement allow to manipulate select web element
Select statement have 3 parameters
* find: CSS selector to identify the select web element
* text: Text to use to selection the option
* wait: optionnal parameter to wait after the statement (in seconds)
Example

```yaml
name: TestSuite Select
testcases:
- name: TestCase Select 
  context:
    type: web
    driver: phantomjs
    debug: true
  steps:
  - action:
      navigate:
        url: https://html.com/tags/select/
  - action:
      select:
        find: article[id='post-289'] select
        text: 'Andean flamingo'
        wait: 1
    screenshot: selectAndean.png
  - action:
      select:
        find: article[id='post-289'] select
        text: 'American flamingo'
    screenshot: selectAmerican.png
```

## Output

* result.url
* result.timeseconds
* result.timehuman
* result.title
* result.find
