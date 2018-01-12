name = "MyTestSuite"

testcase {
  name = "ssh foo status"

  step {
    type = "ssh"
    host = "localhost"
    command = "echo foo"

    assertions = [
      "result.code ShouldEqual 0",
      "result.timeseconds ShouldBeLessThan 10",
    ]
  }
}
