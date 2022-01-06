Feature: HTTP Gherkin test suite on jsonplaceholder.typicode.com and a custom executor

Scenario: HTTP Get posts
    When    HTTP GET https://jsonplaceholder.typicode.com/posts
    Then    result.statuscode ShouldEqual 200
    Then    check post with id 2

Scenario: HTTP Post a new post
    When    create a new post
    Then    result.postjson.id ShouldBeGreaterThan 0
    And     result.postjson.title ShouldBeEmpty