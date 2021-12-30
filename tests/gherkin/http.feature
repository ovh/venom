Feature: HTTP Gherkin test suite on jsonplaceholder.typicode.com 

Scenario: HTTP Get posts
    When    HTTP GET https://jsonplaceholder.typicode.com/posts
    Then    result.statuscode ShouldEqual 200
    Then    HTTP GET https://jsonplaceholder.typicode.com/posts/1
    And     result.bodyjson.id ShouldEqual 1
    And     result.bodyjson.title ShouldNotBeEmpty

Scenario: HTTP Post a new post
    When    HTTP POST https://jsonplaceholder.typicode.com/posts
    Then    result.statuscode ShouldEqual 201