# Venom - Executor Web

Navigate within a web application

Use case: You have a web application and you want to check some behaviors.

The web executor allows you to navigate into your web application and execute actions.

## Web driver

Web driver allow to manipulate a browser.

To use firefox browser, `geckodriver` binary must be installed - see [here](https://github.com/mozilla/geckodriver/releases)
To use chrome browser, `chromedriver` binary must be installed - see [here](https://chromedriver.chromium.org/downloads)

### Firefox
[List of arguments](https://wiki.mozilla.org/Firefox/CommandLineOptions)
[List of preferences](https://searchfox.org/mozilla-central/source/modules/libpref/init/all.js)

### Chrome
[List of arguments](https://peter.sh/experiments/chromium-command-line-switches/)

## Variables

You can define parameters to configure the browser used during the test suite. All parameters are optional:
* width: Width of the browser page
* height: Height of the browser page
* driver (default: `chrome`): Driver to use, severals values possibles:
  * `chrome`: Use chrome driver
  * `gecko`: Use gecko driver (firefox)
* args: Web driver arguments
* prefs: Web driver preferences
* timeout (default: `180`): Timeout in seconds
* debug (default: `false`): Boolean, enabling/disabling the debug mode of the web driver (Deprecated - use logLevel instead)
* proxy: Proxy to use to query web site (Format: `[server]:[port]`)
* headless (default: `false`): Boolean, enabling/disabling the headless mode of the web driver. Usefull to integrate tool in CI/CD pipeline 
* detach (default: `false`): Boolean, enabling/disabling the detach mode of the web driver 
* logLevel (default: `false`): Define log level of client, severals values possibles:
  * `ERROR`: In this mode, you can see errors
  * `WARN`: In this mode, you can see warning
  * `INFO`: In this mode, you can see all methods called
  * `DEBUG`: In this mode, you can see all interactions between web driver and browser
* binaryPath: Binary path of browser, if path is not defined, use default path
* driverPath: Driver path, if path is not defined, use current directory
* driverPort: Port to run driver, if port is not defined, use driver default port

## Action
Action allow to manipulate browser

List of output values available:
* result.url: URL of the current page
* result.timeseconds: duration of the action execution
* result.title: title of the current page

### Navigate
Navigate to a specific URL

#### Input
* URL: Url to navigate
* Reset: Reset browser state

#### Example
```yaml
  - type: web
    action:
      navigate:
        url: https://www.google.fr
```

### Find
Search an element from a selector

[Link for CSS selector tutorial](https://developer.mozilla.org/fr/docs/Web/CSS/CSS_Selectors)
[Link for XPATH selector tutorial](https://developer.mozilla.org/fr/docs/Web/XPath)

#### Input
* Selector: Expression to use to identify web element
* Locator: Locator to use to search element (CSS or XPATH)

#### Output
* result.find: return number of object identified

#### Example
```yaml
  - type: web
    action:
      find: 
        selector: .gsfi
        locator: CSS
    assertions:
    - result.find ShouldEqual 2

  - type: web
    action:
      find:
        selector: //div[@class='FPdoLc lJ9FBc']/center/input[@value='Recherche Google']
        locator: XPATH
    assertions:
    - result.find ShouldEqual 1
    - result.value ShouldEqual Recherche Google
```

### Click
Click on an element from a selector

#### Input
* Find: Element to find (More informations in find section)
* Wait: Time to wait after click (in seconds)
* SyncTimeout: Option to enable element synchronization (wait until element appear in page). SyncTimeout allow to define maximum time to wait for synchronization

#### Example
```yaml
  - type: web
    action:
      click:
        find:
          selector: button
          locator: CSS
```

### Fill
Fill an element (input or textarea) with a text

#### Input
Array of structured input:
* Find: Element to find (More informations in find section)
* Text: Update element value with text
* Key: Update element value with key
* SyncTimeout: Option to enable element synchronization (wait until element appear in page). SyncTimeout allow to define maximum time to wait for synchronization

#### Example
```yaml
  - type: web
    action:
      fill:
      - find:
          selector: input
          locator: CSS
        text: userName
```

### Select
Select an option of a web element

#### Input
* Find: Element to find (More informations in find section)
* Text: Option to select
* Wait: Time to wait after click (in seconds)
* SyncTimeout: Option to enable element synchronization (wait until element appear in page). SyncTimeout allow to define maximum time to wait for synchronization

#### Example
```yaml
  - type: web
    action:
      select:
        find:
          selector: select
          locator: CSS
        text: Option 1
```

### Upload file
Upload file on a web element

#### Input
* Find: Element to find (More informations in find section)
* Files: String array to define files to upload
* Wait: Time to wait after click (in seconds)
* SyncTimeout: Option to enable element synchronization (wait until element appear in page). SyncTimeout allow to define maximum time to wait for synchronization

#### Example
```yaml
  - type: web
    action:
      uploadFile:
        find:
          selector: input
          locator: CSS
        files: 
        - toUpload.csv
```

### Wait
Wait time
An integer value to define number of seconds to wait

#### Example
```yaml
  - type: web
    action:
      wait: 1
```

### Select frame
Select a frame presents in the current page

#### Input
* Find: Frame to find (More informations in find section)
* SyncTimeout: Option to enable element synchronization (wait until element appear in page). SyncTimeout allow to define maximum time to wait for synchronization

#### Example
```yaml
  - type: web
    action:
      selectFrame:
        find:
          selector: "#iframeResult"
          locator: CSS
```

### Select root frame
Select root frame presents in the current page
A boolean value to set to true to select root frame

#### Example
```yaml
  - type: web
    action:
      selectRootFrame: true
```

### Next window
Select the next window
A boolean value to set to true to select next window

#### Example
```yaml
  - type: web
    action:
      nextWindow: true
```

### Back
Back to previous page
A boolean value to set to true to back to previous page

#### Example
```yaml
  - type: web
    action:
      historyAction: back
    assertions:
    - result.title ShouldStartWith GitHub
    - result.url ShouldEqual https://github.com/
```

### Forward
Foward to next page
A boolean value to set to true to forward page

#### Example
```yaml
  - type: web
    action:
      historyAction: forward
    assertions:
    - result.title ShouldStartWith GitHub
    - result.url ShouldEqual https://github.com/team
```

### Refresh
Refresh page
A boolean value to set to true to refresh page

#### Example
```yaml
  - type: web
    action:
      historyAction: refresh
    assertions:
    - result.title ShouldStartWith GitHub
    - result.url ShouldEqual https://github.com/team
```

### Confirm popup
Confirm popup dialog (confirm dialog box and alert dialog box)
A boolean value to set to true to confirm popup dialog

#### Example
```yaml
  - type: web
    action:
      confirmPopup: true
```

### Cancel popup
Cancel popup dialog
A boolean value to set to true to cancel popup dialog

#### Example
```yaml
  - type: web
    action:
      cancelPopup: true
```

### Execute javascript
Execute javascript code

#### Input
* Command: Javascript code to execute
* Args: String array to define javascript arguments

#### Example
```yaml
  - type: web
    action:
      execute:
        command: "window.editor.setValue(\"<!DOCTYPE html> <html></html>\"); window.editor.save();"
```


More informations about actions (https://github.com/ovh/venom/tree/master/executors/web/types.go)
For an action, you can take screenshot of browser with following command: `screenshot: [fileName].png`

## Example
A global example

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
