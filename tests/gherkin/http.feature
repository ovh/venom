Feature: HTTP Gherkin test suite

Scenario: HTTP Get test
    When    HTTP Get https://eu.api.ovh.com/1.0/
    Then    Body ShouldContainSubstring /dedicated/server
    And     Body ShouldContainSubstring /ipLoadbalancing
    And     Statuscode ShouldEqual 200
    And     Bodyjson.apis.apis0.path ShouldEqual /allDom

Scenario: HTTP Post test
    When    HTTP Get https://eu.api.ovh.com/1.0/
    Then    Body ShouldContainSubstring /dedicated/server
    And     Body ShouldContainSubstring /ipLoadbalancing
    And     Statuscode ShouldEqual 200
    And     Bodyjson.apis.apis0.path ShouldEqual /allDom