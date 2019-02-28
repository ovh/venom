// +build yaml

package main

const expected = `TAP version 13
1..2
ok 1 - test for anchoring the YAML block
  ---
  code: 3
  message: testing YAML blocks
  ...
`
