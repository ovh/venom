# Venom - Executor Web

Navigate in a web application

Use case: You have a web application and you want to check some behaviours?
Venom allows you to navigate into it and execute actions.

## Input

* Action (https://github.com/ovh/venom/tree/master/executors/web/types.go)
* Format

Web context allows you to configure the browser used for navigation. All parameters are optional:
* width: Width of the browser page
* height: Height of the browser page
* driver: `chrome`, `gecko` or `phantomjs` (default: `phantomjs`)
* args: List of arguments for `chrome` driver (see [here](https://peter.sh/experiments/chromium-command-line-switches/))
* prefs: List of user preferences for `chrome` driver, using dot notation (see [here](http://www.chromium.org/administrators/configuring-other-preferences) and [here](https://src.chromium.org/viewvc/chrome/trunk/src/chrome/common/pref_names.cc?view=markup))
* timeout: Timeout in seconds (default: 180)
* debug: Boolean enabling the debug mode of the web driver (default: false)

```yaml
name: TestSuite Web
testcases:
- name: TestCase Google search
  context:
    type: web
    width: 1920
    height: 1080
    driver: phantomjs
    args:
    - 'browser-test'
    prefs:
      profile.default_content_settings.popups: 0
      profile.default_content_setting_values.notifications: 1
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

Upload file actiow allow you to upload file with file input web component
Example:

```yaml
name: TestSuiteUploadFile
testcases:
- name: TestCaseUploadFile
  context:
    type: web
    driver: chrome
    debug: true
  steps:
  - action:
      navigate:
        url: https://www.w3schools.com/tags/tryit.asp?filename=tryhtml5_input_type_file
  - action:
      selectFrame:
        find: iframe[id='iframeResult']
  - action:
      uploadFile:
        find: form:nth-child(3) input#myfile
        files:
        - myFile.png
    screenshot: "result.png"
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

History actions allow you to manipulate browser history actions. The following actions are available:
* back
* refresh
* forward

Example:

```yaml
name: TestSuiteNavigationHistory
testcases:
- name: TestCaseNavigationHistory
  context:
    type: web
    driver: chrome
    debug: true
  steps:
  - action:
      navigate:
        url: https://www.google.com
  - action:
        fill:
        - find: input[name='q']
          text: ovh venom github
    screenshot: search.png
  - action:
        click:
            find: div[jsname='VlcLAe'] input[name='btnK']
            wait: 2
  - action:
        historyAction: back
  - action:
        historyAction: refresh
  - action:
        historyAction: forward
```

## Output

* result.url
* result.timeseconds
* result.title
* result.find


## Chrome
This section describes some features specific to the Chrome browser

### CI
If you want to include Chrome Driver tests in your integration pipeline, you must execute Chrome in headless mode.

Example
```yaml
name: TestSuite Web
testcases:
- name: Test disable same site security
  context:
    type: web
    width: 1920
    height: 1080
    driver: chrome
    args:
    - 'headless'
    timeout: 60
    debug: true
  steps:
  - action:
      navigate:
        url: https://www.google.fr
```


### Flags

In Chrome, you can turn experimental features on or off to test the behavior of upcoming features.
You can do this manually with chrome://flags url with Chrome browser.
In Venom, to enable a feature, add an instance of the enable-features argument.
To disable a feature, add a disable-features argument instance

Example
```yaml
name: TestSuite Web
testcases:
- name: Test disable same site security
  context:
    type: web
    width: 1920
    height: 1080
    driver: chrome
    args:
    - 'disable-features=SameSiteByDefaultCookies'
    - 'enable-features=CookiesWithoutSameSiteMustBeSecure'
    timeout: 60
    debug: true
  steps:
  - action:
      navigate:
        url: https://samesite-sandbox.glitch.me/
  - action:
      wait: 5
```
