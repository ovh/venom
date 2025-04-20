package playwright

import (
	"testing"
)

func TestParseValidActionLines(t *testing.T) {
	testCases := []struct {
		ActionLine string
		Action     string // expected action
	}{
		{ActionLine: `Click "some-element"`, Action: "Click"},
		{ActionLine: `DoubleClick "some-element"`, Action: "DoubleClick"},
		{ActionLine: `Doubleclick "some-element"`, Action: "Doubleclick"},
		{ActionLine: `Tap "some-element"`, Action: "Tap"},
		{ActionLine: `Focus "some-element"`, Action: "Focus"},
		{ActionLine: `Blur "some-element"`, Action: "Blur"},
		{ActionLine: `Clear "some-element"`, Action: "Clear"},
		{ActionLine: `Check "some-element"`, Action: "Check"},
		{ActionLine: `Uncheck "some-element"`, Action: "Uncheck"},
		{ActionLine: `Fill "some-element" "some text"`, Action: "Fill"},
		{ActionLine: `Press "some-element"`, Action: "Press"},
		{ActionLine: `Type "some-element"`, Action: "Type"},
	}

	for _, tc := range testCases {
		if action, _, _, err := parseActionLine(tc.ActionLine); err != nil {
			t.Errorf("failed to parse action, got error '%v'", err)
		} else {
			if action != tc.Action {
				t.Errorf("failed to parse action, expected '%s' but got '%s'", tc.Action, action)
			}
		}
	}
}

func TestParseInvalidActionLines(t *testing.T) {
	testCases := []string{
		``,
		`Click`,
		`Clickit "some-element"`,
		`Double-Click "some-element"`,
		`Double click "some-element"`,
		`Taps"some-element"`,
		`Focuses "some-element"`,
		`Blurs "some-element"`,
		`Cleare "some-element"`,
		`Checked "some-element"`,
		`Unchecked "some-element"`,
		`Pressed "some-element"`,
		`Typed "some-element"`,
	}

	for _, tc := range testCases {
		if _, _, _, err := parseActionLine(tc); err == nil {
			t.Errorf("failed to parse action, expected error but got none for '%s'", tc)
		}
	}
}
