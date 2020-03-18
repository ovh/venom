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

Select frame and Select root frame actions help you to navigate into your differents frames.
After the frame selection, you can manipulate web elements presents in a frame.
Two statements:
* SelectFrame: One find parameter to select the frame with CSS selector
* SelectRootFrame: One boolean parameter, must be true to activate the statement
Example:

```yaml
name: TestSuite SelectFrame
testcases:
- name: TestCase SelectFrame 
  context:
    type: web
    driver: phantomjs
    debug: true
  steps:
  - action:
      navigate:
        url: https://www.w3schools.com/jsref/tryit.asp?filename=tryjsref_win_open
  - action:
      selectFrame:
        find: iframe[id='iframeResult']
  - action:
      find: body > button
    assertions:
    - result.find ShouldEqual 1
  - action:
      find: a#tryhome
    assertions:
    - result.find ShouldEqual 0
  - action:
      selectRootFrame: true
  - action:
      find: body > button
    assertions:
    - result.find ShouldEqual 0
  - action:
      find: a#tryhome
    assertions:
    - result.find ShouldEqual 1
```

Next Window action allow you to change the current window
Next Window have one boolean parameter, this parameter must be true
Example:

```yaml
name: TestSuite NextWindow
testcases:
- name: TestCase NextWindow 
  context:
    type: web
    driver: chrome
    debug: true
  steps:
  - action:
      navigate:
        url: https://javascript.info/popup-windows
  - action:
      click:
        find: article > div:nth-child(3) > div:nth-child(17) a[data-action='run']
        wait: 4
    screenshot: beforeNextWindow.png
  - action:
      nextWindow: true
    screenshot: resultNextWindow.png
    assertions:
      - result.url ShouldStartWith https://www.google.com
```

## Output

* result.url
* result.timeseconds
* result.timehuman
* result.title
* result.find
