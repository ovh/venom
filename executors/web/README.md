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

ConfirmPopup and CancelPopup actions allow you to manipulate modal dialog initialized by the alert and confirm javascript statement.
These two actions have one boolean parameter and the parameter value must be true to activate the action.
These actions are not compatible with PhantomJS driver.

Example:

```yaml
name: TestSuite Popup
testcases:
- name: TestCase Popup 
  context:
    type: web
    driver: chrome
    debug: true
  steps:
  - action:
      navigate:
        url: https://javascript.info/alert-prompt-confirm
  - action:
      click:
        find: article > div:nth-child(3) > div:nth-child(8) a[data-action='run']
        wait: 1
  - action:
      ConfirmPopup: true
  - action:
      click:
        find: article > div:nth-child(3) > div:nth-child(26) a[data-action='run']
        wait: 1
  - action:
      ConfirmPopup: true
  - action:
      ConfirmPopup: true
  - action:
      click:
        find: article > div:nth-child(3) > div:nth-child(26) a[data-action='run']
        wait: 1
  - action:
      CancelPopup: true
  - action:
      ConfirmPopup: true
```

## Output

* result.url
* result.timeseconds
* result.timehuman
* result.title
* result.find
