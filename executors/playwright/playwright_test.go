package playwright

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ovh/venom"
	playwrightgo "github.com/playwright-community/playwright-go"
)

const testPage = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Test Page</title>
</head>

<body>
    <form method="post" action="https://example.com/submit" id="example-form">
        <h1>Example form</h1>
        <div class="control"><input type="text" name="firstName" id="firstName"></div>
        <div class="control"><input type="text" name="lastName" id="lastName"></div>
        <div class="control"><input type="text" name="age" id="age"></div>
        <div class="control"><input type="email" name="email" id="email"></div>
        <div class="control"><textarea name="bio" id="biography"></textarea></div>
        <button id="submit-button" type="submit">Submit</button>
    </form>
    <div id="age-shown-when-input" style="display: none;">
        <h4>AGE</h4>
        <span id="inputted-age"></span>
    </div>

    <script>
        document.addEventListener('DOMContentLoaded', function () {

            const el = document.getElementById('age');
            age.addEventListener('input', function () {
                const ageValue = el.value;
                const ageShown = document.getElementById('inputted-age');
                ageShown.textContent = ageValue;
                ageShown.parentElement.style.display = ageValue ? 'block' : 'none';
            });
        });
    </script>
</body>
</html>
`

func TestPerformActions(t *testing.T) {
	venom.InitTestLogger(t)

	testActions := []ExecutorAction{
		{Action: "Fill", Selector: "#firstName", Content: "John"},
		{Action: "Fill", Selector: "[name=lastName]", Content: "John"},
		{Action: "Fill", Selector: "#age", Content: "24"},
		{Action: "Focus", Selector: "#email"},
		{Action: "WaitFor", Selector: "#inputted-age", Content: "24"},
		{Action: "Click", Selector: "#submit-button"},
	}

	pw, err := playwrightgo.Run()
	if err != nil {
		t.Fail()
	}
	browser, err := pw.Chromium.Launch(playwrightgo.BrowserTypeLaunchOptions{
		Headless: playwrightgo.Bool(true),
	})
	if err != nil {
		t.Fail()
	}
	browserCtx, err := browser.NewContext()
	if err != nil {
		t.Fail()
	}
	page, err := browserCtx.NewPage()
	if err != nil {
		t.Fail()
	}

	err = page.SetContent(testPage, playwrightgo.PageSetContentOptions{})
	if err != nil {
		t.Error("failed to set testPage content")
	}

	err = performActions(context.Background(), page, testActions)
	if err != nil {
		t.Errorf("failed to test actions %v", err)
	}

	t.Cleanup(func() {
		err = browser.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to close browser properly %v", err)
		}
		err = pw.Stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to close browser properly %v", err)
		}
	})
}
