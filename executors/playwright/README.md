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
    url: https://parrotanything.com
    auto-install: true
    browser: chrome # options are: chromium|chrome|firefox
    assertions:
    - page.body ShouldContainSubstring NNDI
    - page.body.$(".h1:first") ShouldEqual Parrot

```
