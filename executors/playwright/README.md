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
      url: https://news.ycombinator.com
      assertions:
        - result.page.body ShouldContainSubstring Hacker News
        - result.document.body ShouldContainSubstring Hacker News

```
