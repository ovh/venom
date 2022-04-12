# Venom - Executor Web

Navigate within a web application

Use case: You have a web application and you want to check some behaviors.

The web executor allows you to navigate into your web application and execute actions.

## Input

* Action (https://github.com/ovh/venom/tree/master/executors/web/types.go)
* `screenshot` allows to dump the display of the current web page in a .png file


You can define parameters to configure the browser used during the test suite. All parameters are optional:
* width: Width of the browser page
* height: Height of the browser page
* driver (default: `phantomjs`): 
  * `phantomjs` (the `phantomjs` binary must be installed - see [here](https://phantomjs.org/))
  * `gecko` (the `geckodriver` binary must be installed - see [here](https://github.com/mozilla/geckodriver/releases)) 
  * `chrome` (the `chromedriver` binary must be installed - see [here](https://chromedriver.chromium.org/downloads)) 
* args: List of arguments for `chrome` driver (see [here](https://peter.sh/experiments/chromium-command-line-switches/))
* prefs: List of user preferences for `chrome` driver, using dot notation (see [here](http://www.chromium.org/administrators/configuring-other-preferences) and [here](https://src.chromium.org/viewvc/chrome/trunk/src/chrome/common/pref_names.cc?view=markup))
* timeout: Timeout in seconds (default: `180`)
* debug: Boolean, enabling/disabling the debug mode of the web driver (default: `false`)


```yaml
name: TestSuite Web
vars:
  web:
    driver: chrome
    width: 1920
    height: 1080
    args:
    - 'browser-test'
    prefs:
      profile.default_content_settings.popups: 0
      profile.default_content_setting_values.notifications: 1
    timeout: 60
    debug: true
testcases:
- name: TestCase Google search
  steps:
  - type: web
    action:
      navigate:
        url: https://www.google.fr
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
      - find: input[name="q"]
        text: "venom ovh"
  - type: web
    action:
      click:
        find: input[value="Recherche Google"]
        wait: 1
    screenshot: googlesearch.png

```


The `selectFrame` and `selectRootFrame` actions allow to navigate into the different frames of the web page.
After the frame selection, you can manipulate web elements present in this frame.

`selectFrame` has one parameter (`find`) to select the frame with its CSS selector.

`selectRootFrame` has one boolean parameter which must be set to `true` to activate the action.

Example:

```yaml
name: TestSuite SelectFrame
vars:
  web:
    driver: phantomjs
    debug: true
testcases:
- name: TestCase SelectFrame 
  steps:
  - type: web
    action:
      navigate:
        url: https://www.w3schools.com/jsref/tryit.asp?filename=tryjsref_win_open
  - type: web
    action:
      selectFrame:
        find: iframe[id='iframeResult']
  - type: web
    action:
      find: body > button
    assertions:
    - result.find ShouldEqual 1
  - type: web
    action:
      find: a#tryhome
    assertions:
    - result.find ShouldEqual 0
  - type: web
    action:
      selectRootFrame: true
  - type: web
    action:
      find: body > button
    assertions:
    - result.find ShouldEqual 0
  - type: web
    action:
      find: a#tryhome
    assertions:
    - result.find ShouldEqual 1
```

The `nextWindow` action allows to change the current window. This action has one boolean parameter which must be set to `true` to activate the action.

Example:

```yaml
name: TestSuite NextWindow
vars:
  web:
    driver: chrome
    debug: true
testcases:
- name: TestCase NextWindow 
  steps:
  - type: web
    action:
      navigate:
        url: https://javascript.info/popup-windows
  - type: web
    action:
      click:
        find: article > div:nth-child(3) > div:nth-child(17) a[data-action='run']
        wait: 4
    screenshot: beforeNextWindow.png
  - type: web
    action:
      nextWindow: true
    screenshot: resultNextWindow.png
    assertions:
      - result.url ShouldStartWith https://www.google.com
```

The `uploadFile` action allows to upload a file into a web page.

Example:

```yaml
name: TestSuiteUploadFile
vars:
  web:
    driver: chrome
    debug: true
testcases:
- name: TestCaseUploadFile
  steps:
  - type: web
    action:
      navigate:
        url: https://www.w3schools.com/tags/tryit.asp?filename=tryhtml5_input_type_file
  - type: web
    action:
      selectFrame:
        find: iframe[id='iframeResult']
  - type: web
    action:
      uploadFile:
        find: form:nth-child(3) input#myfile
        files:
        - myFile.png
    screenshot: result.png
```

The `select` action allows to select an item into a list (a select web element) in a web page.

This action has 3 parameters:

* `find`: the CSS selector to identify the select web element
* `text`: the item to select in the list
* `wait`: (optional) pause after the selection is made

Example:

```yaml
name: TestSuite Select
vars:
  web:
    driver: phantomjs
    debug: true
testcases:
- name: TestCase Select 
  steps:
  - type: web
    action:
      navigate:
        url: https://html.com/tags/select/
  - type: web
    action:
      select:
        find: article[id='post-289'] select
        text: 'Andean flamingo'
        wait: 1
    screenshot: selectAndean.png
  - type: web
    action:
      select:
        find: article[id='post-289'] select
        text: 'American flamingo'
    screenshot: selectAmerican.png
```

The `confirmPopup` and `cancelPopup` actions allow to manipulate modal dialog boxes displayed by the alert and the confirm javascript statements.

Both actions have one boolean parameter which must be set to `true` to activate the action. 

Warning: These actions are not compatible with the `phantomJS` driver.

Example:

```yaml
name: TestSuite Popup
vars:
  web:
    driver: chrome
    debug: true
testcases:
- name: TestCase Popup 
  steps:
  - type: web
    action:
      navigate:
        url: https://javascript.info/alert-prompt-confirm
  - type: web
    action:
      click:
        find: article > div:nth-child(3) > div:nth-child(8) a[data-action='run']
        wait: 1
  - type: web
    action:
      ConfirmPopup: true
  - type: web
    action:
      click:
        find: article > div:nth-child(3) > div:nth-child(26) a[data-action='run']
        wait: 1
  - type: web
    action:
      ConfirmPopup: true
  - type: web
    action:
      ConfirmPopup: true
  - type: web
    action:
      click:
        find: article > div:nth-child(3) > div:nth-child(26) a[data-action='run']
        wait: 1
  - type: web
    action:
      CancelPopup: true
  - type: web
    action:
      ConfirmPopup: true
```

The `historyAction` action is provided to manage the browser history.

This action has three possible values:
* `back`
* `refresh`
* `forward`

Example:

```yaml
name: TestSuiteNavigationHistory
vars:
  web:
    driver: chrome
    debug: true
testcases:
- name: TestCaseNavigationHistory
  steps:
  - type: web
    action:
      navigate:
        url: https://www.google.com
  - type: web
    action:
        fill:
        - find: input[name='q']
          text: ovh venom github
    screenshot: search.png
  - type: web
    action:
        click:
            find: div[jsname='VlcLAe'] input[name='btnK']
            wait: 2
  - type: web
    action:
        historyAction: back
  - type: web
    action:
        historyAction: refresh
  - type: web
    action:
        historyAction: forward
```

## Output

* result.url: URL of the current page
* result.timeseconds: duration of the action execution
* result.title: title of the current page
* result.find: equals to 1 if the requested web element has been found


## Chrome
This section describes some features specific to the Chrome browser.

### CI
If you want to include some Chrome Driver tests in your integration pipeline, you must execute Chrome in headless mode.

Example
```yaml
name: TestSuite Web
vars:
  web:
    width: 1920
    height: 1080
    driver: chrome
    args:
    - 'headless'
    timeout: 60
    debug: true
testcases:
- name: Test disable same site security
  steps:
  - type: web
    action:
      navigate:
        url: https://www.google.fr
```

### Flags

In the Chrome browser, you can turn experimental features on or off to test the behavior of upcoming features (check the chrome://flags url).

In the web executor, to enable a feature, use the `enable-features` argument.

To disable a feature, use the `disable-features` argument.

Example
```yaml
name: TestSuite Web
vars:
  web:
    driver: chrome
    args:
    - 'disable-features=SameSiteByDefaultCookies'
    - 'enable-features=CookiesWithoutSameSiteMustBeSecure'
testcases:
- name: Test disable same site security
  steps:
  - type: web
    action:
      navigate:
        url: https://samesite-sandbox.glitch.me/
  - type: web
    action:
      wait: 5
```
