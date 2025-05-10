# Plawright Executor

The playwright executor allows you yo use venom to run Playwright tests
with the same yaml configuration file.

> NOTE: the playwright executor needs to use Playwright and as a result, will
> attempt to install playwright if it is not already installed.
> We use the [playwright-go](https://github.com/playwright-community/playwright-go) library for this

```yaml
name: Playwright testsuite
testcases:
- name: Check the title
  steps:
    - type: playwright
      url: http://localhost:5173/
      headless: true
      actions:
        - action: Fill
          selector: "#email"
          content: "change@example.com"
        # you can write the expression in one line like this, if you want
        - { action: Fill, selector: "#email", content: "zikani@example.com" }
        - action: Fill
          selector: "#password"
          content: "zikani123"
        - action: Click
          selector: "#loginButton"
        - action: WaitFor
          selector: ".second-dashboard-user-name"
      assertions:
        - result.page.body ShouldContainSubstring Parrot
        - result.document.body ShouldContainSubstring Hello,&nbsp;Zikani
        - result.document.body ShouldContainSubstring Logout
```


## Available actions

|Action|Arguments|Example|
|------|---------|-------|
|Click                 |**querySelector**| Click "#element" |
|DoubleClick           |**querySelector**| DoubleClick "#element" |
|Tap                   |**querySelector**| Tap "#element" |
|Focus                 |**querySelector**| Focus "#element" |
|Blur                  |**querySelector**| Blur "#element" |
|Fill                  |**querySelector** TEXT| Fill "#element" "my input text" |
|Clear                 |**querySelector**| Clear "#element" |
|Check                 |**querySelector**| Check "#element" |
|Uncheck               |**querySelector**| Uncheck "#element" |
|FillCheckbox          |**querySelector**| FillCheckbox "#element" |
|Press                 |**querySelector** TEXT| Press "#element" "some text"|
|PressSequentially     |**querySelector** TEXT | PressSequentially "#element" "some input"|
|Type                  |**querySelector** TEXT | Type "#element" |
|Select                |**querySelector** TEXT | Select "#someSelect" "Value or Label"|
|SelectOption          |**querySelector** TEXT | Select "#someSelect" "Value or Label"|
|SelectMultipleOptions |**querySelector** TEXT | SelectMultipleOptions "#someSelect" "Value or Label 1,Value or Label 2,..., Value or Label N"|
|WaitFor               |**querySelector**| WaitFor "#element" |
|WaitForSelector       |**querySelector**| WaitForSelector "#element" |
|Goto                  |**REGEX**| Goto "^some-page" |
|WaitForURL            |**REGEX**| WaitForURL "^some-page" |
|GoBack                |N/A| GoBack |
|GoForward             |N/A| GoForward |
|Refresh               |N/A| Refresh |
