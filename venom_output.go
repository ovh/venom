package venom

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/fatih/color"
	tap "github.com/mndrix/tap-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	color.NoColor = true
	if os.Getenv("IS_TTY") == "" || strings.ToLower(os.Getenv("IS_TTY")) == "true" || os.Getenv("IS_TTY") == "1" {
		color.NoColor = false
	}
}

// OutputResult output result to sdtout, files...
func (v *Venom) OutputResult(elapsed time.Duration) error {
	if v.OutputDir == "" {
		return nil
	}
	for i := range v.testsuites {
		tcFiltered := []TestCase{}
		for _, tc := range v.testsuites[i].TestCases {
			if tc.IsEvaluated {
				tcFiltered = append(tcFiltered, tc)
			}
		}
		v.testsuites[i].TestCases = tcFiltered

		testsResult := &Tests{}
		computeStats(testsResult, &v.testsuites[i])

		var data []byte
		var err error
		switch v.OutputFormat {
		case "json":
			data, err = json.MarshalIndent(testsResult, "", "  ")
			if err != nil {
				log.Fatalf("Error: cannot format output json (%s)", err)
			}
		case "tap":
			data, err = outputTapFormat(*testsResult)
			if err != nil {
				log.Fatalf("Error: cannot format output tap (%s)", err)
			}
		case "yml", "yaml":
			data, err = yaml.Marshal(testsResult)
			if err != nil {
				log.Fatalf("Error: cannot format output yaml (%s)", err)
			}
		default:
			dataxml, errm := xml.MarshalIndent(testsResult, "", "  ")
			if errm != nil {
				log.Fatalf("Error: cannot format xml output: %s", errm)
			}
			data = append([]byte(`<?xml version="1.0" encoding="utf-8"?>`), dataxml...)
		}

		filename := path.Join(v.OutputDir, "test_results."+v.testsuites[i].Filename+"."+v.OutputFormat)
		if err := os.WriteFile(filename, data, 0600); err != nil {
			return fmt.Errorf("Error while creating file %s: %v", filename, err)
		}
		v.PrintFunc("Writing file %s\n", filename)
	}

	if v.OutputFormat == "html" {
		testsResult := &Tests{}
		for i := range v.testsuites {
			computeStats(testsResult, &v.testsuites[i])
		}

		data, err := outputHTML(testsResult)
		if err != nil {
			log.Fatalf("Error: cannot format output html (%s)", err)
		}
		filename := path.Join(v.OutputDir, "test_results.html")
		if err := os.WriteFile(filename, data, 0600); err != nil {
			return fmt.Errorf("Error while creating file %s: %v", filename, err)
		}
	}
	return nil
}

func computeStats(testsResult *Tests, ts *TestSuite) {
	testsResult.TestSuites = append(testsResult.TestSuites, *ts)
	if ts.Failures > 0 || ts.Errors > 0 {
		testsResult.TotalKO++
	} else {
		testsResult.TotalOK++
	}
	if ts.Skipped > 0 {
		testsResult.TotalSkipped++
	}

	testsResult.Total = testsResult.TotalKO + testsResult.TotalOK + testsResult.TotalSkipped
}

func outputTapFormat(tests Tests) ([]byte, error) {
	tapValue := tap.New()
	buf := new(bytes.Buffer)
	tapValue.Writer = buf
	tapValue.Header(tests.Total)
	for _, ts := range tests.TestSuites {
		for _, tc := range ts.TestCases {
			name := ts.Name + " / " + tc.Name
			if len(tc.Skipped) > 0 {
				tapValue.Skip(1, name)
				continue
			}

			if len(tc.Errors) > 0 {
				tapValue.Fail(name)
				for _, e := range tc.Errors {
					tapValue.Diagnosticf("Error: %s", e.Value)
				}
				continue
			}

			if len(tc.Failures) > 0 {
				tapValue.Fail(name)
				for _, e := range tc.Failures {
					tapValue.Diagnosticf("Failure: %s", e.Value)
				}
				continue
			}

			tapValue.Pass(name)
		}
	}

	return buf.Bytes(), nil
}

const templateHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Venom Results</title>

    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.0/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-gH2yIJqKdNHPEq0n4Mqa/HGKIhSkIHeL5AyhkYV8i59U5AR6csBvApHHNl/vI1Bx" crossorigin="anonymous">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/jquery/3.6.0/jquery.min.js" integrity="sha512-894YE6QWD5I59HgZOGReFYm4dnWc1Qt5NtvYSaNcOP+u1T9qYdvdihz0PPSiiqn/+/3e7Jo4EaG7TubfWGUrMQ==" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.0/dist/js/bootstrap.bundle.min.js" integrity="sha384-A3rJD856KowSb7dwlZdYEkO39Gagi7vIsF0jrRAoQmDKKtQBHUuLZ9AsSv4jD4Xa" crossorigin="anonymous"></script>


    <style>
    .bd-placeholder-img {
      font-size: 1.125rem;
      text-anchor: middle;
      -webkit-user-select: none;
      -moz-user-select: none;
      user-select: none;
    }

    @media (min-width: 768px) {
      .bd-placeholder-img-lg {
        font-size: 3.5rem;
      }
    }

    .b-example-divider {
      height: 3rem;
      background-color: rgba(0, 0, 0, .1);
      border: solid rgba(0, 0, 0, .15);
      border-width: 1px 0;
      box-shadow: inset 0 .5em 1.5em rgba(0, 0, 0, .1), inset 0 .125em .5em rgba(0, 0, 0, .15);
    }

    .b-example-vr {
      flex-shrink: 0;
      width: 1.5rem;
      height: 100vh;
    }

    .bi {
      vertical-align: -.125em;
      fill: currentColor;
    }

    .nav-scroller {
      position: relative;
      z-index: 2;
      height: 2.75rem;
      overflow-y: hidden;
    }

    .nav-scroller .nav {
      display: flex;
      flex-wrap: nowrap;
      padding-bottom: 1rem;
      margin-top: -1px;
      overflow-x: auto;
      text-align: center;
      white-space: nowrap;
      -webkit-overflow-scrolling: touch;
    }

    body {
      font-size: .875rem;
    }
    
    .feather {
      width: 16px;
      height: 16px;
    }
    
    /*
     * Sidebar
     */
    
    .sidebar {
      position: fixed;
      top: 0;
      /* rtl:raw:
      right: 0;
      */
      bottom: 0;
      /* rtl:remove */
      left: 0;
      z-index: 100; /* Behind the navbar */
      padding: 48px 0 0; /* Height of navbar */
      box-shadow: inset -1px 0 0 rgba(0, 0, 0, .1);
    }
    
    @media (max-width: 767.98px) {
      .sidebar {
        top: 5rem;
      }
    }
    
    .sidebar-sticky {
      height: calc(100vh - 48px);
      overflow-x: hidden;
      overflow-y: auto; /* Scrollable contents if viewport is shorter than content. */
    }
    
    .sidebar .nav-link {
      font-weight: 500;
      color: #333;
    }
    
    .sidebar .nav-link .feather {
      margin-right: 4px;
      color: #727272;
    }
    
    .sidebar .nav-link.active {
      color: #2470dc;
    }
    
    .sidebar .nav-link:hover .feather,
    .sidebar .nav-link.active .feather {
      color: inherit;
    }
    
    .sidebar-heading {
      font-size: .75rem;
    }
    
    /*
     * Navbar
     */
    
    .navbar-brand {
      padding-top: .75rem;
      padding-bottom: .75rem;
      background-color: rgba(0, 0, 0, .25);
      box-shadow: inset -1px 0 0 rgba(0, 0, 0, .25);
    }
    
    .navbar .navbar-toggler {
      top: .25rem;
      right: 1rem;
    }
    
    .navbar .form-control {
      padding: .75rem 1rem;
    }
    
    .form-control-dark {
      color: #fff;
      background-color: rgba(255, 255, 255, .1);
      border-color: rgba(255, 255, 255, .1);
    }
    
    .form-control-dark:focus {
      border-color: transparent;
      box-shadow: 0 0 0 3px rgba(255, 255, 255, .25);
    }

    .float-right {
      float: right!important;
    }

  </style>

  </head>
<body>
    
<header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
  <a class="navbar-brand col-md-3 col-lg-2 me-0 px-3 fs-6" href="#">🐍 Venom</a>
  <div class="navbar-nav">
  </div>
</header>

<div class="container-fluid">
  <div class="row">
    <nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse">
      <div class="position-sticky pt-3 sidebar-sticky">
        
        <h6 class="sidebar-heading d-flex justify-content-between align-items-center px-3 mt-4 mb-1 text-muted text-uppercase">
          <span>Tests Suites</span>
        </h6>
        <ul class="nav flex-column mb-2" id="testsuites">
        </ul>
      </div>
    </nav>

    <main class="col-md-9 ms-sm-auto col-lg-10 px-md-4">
      <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pt-3 pb-2 mb-3 border-bottom">
        <h1 id="testsuite" class="h2"></h1>

        <div class="d-flex text-decoration-none align-items-center">
          <span id="totalsuiteSkipped" class="badge rounded-pill float-right text-bg-dark" title="skipped"></span>
          <span id="totalsuiteErrors" class="badge rounded-pill float-right text-bg-danger" title="errors"></span>
          <span id="totalsuitefailures" class="badge rounded-pill float-right text-bg-danger" title="failures"></span>
          <span id="totalsuiteSuccess" class="badge rounded-pill float-right text-bg-success" title="ok"></span>
        </div>
        
      </div>

      <h3 id="totalsuiteCases"></h3>

      <div class="accordion" id="testcases">
        
      </div>

    </main>
  </div>
</div>

<script>
  (() => {
    'use strict'

    $(document).ready( function () {
      var a = {{.JSONValue}};

      dis("#totalDisabled", a.disabled, "disabled", "disabled")
      dis("#totalSkipped", a.skipped, "skipped", "skipped")
      dis("#totalErrors", a.ko, "error", "errors")
      dis("#totalSuccess", a.ok, "success", "success")

      var testsuites = "";

      for (var i = 0; i < a.test_suites.length; i++) { 
        if (a.test_suites[i]) {
          testsuites += navbartestsuites(i, a.test_suites[i]); 
        }
      }

      $('#testsuites').html(testsuites)

      $('#testsuites li a').on('click', function () {
        var data = a.test_suites[this.id];
        $('#testsuite').html(data.name);

        dis("#totalsuiteCases", data.total, "testcase", "testcases")
        dis("#totalsuiteSkipped", data.skipped, "skipped", "skipped")
        dis("#totalsuiteDisabled", data.disabled, "disabled", "disabled")
        dis("#totalsuiteErrors", data.errors, "error", "errors")
        dis("#totalsuiteErrors", data.errors, "error", "errors")
        dis("#totalsuitefailures", data.failures, "failure", "failures")
        var ok = data.total - data.errors - data.disabled - data.failures - data.skipped;
        dis("#totalsuiteSuccess", ok, "success", "success")

        middletestsuite(i, data);
        
      });
    });

    const collapseElementList = document.querySelectorAll('.collapse')
    const collapseList = [...collapseElementList].map(collapseEl => new bootstrap.Collapse(collapseEl))


  })()

  function navbartestsuites(idx, testsuite) {
    var badges = "";
    if (testsuite.disabled > 0) {
        badges += '<span class="badge rounded-pill float-right text-bg-dark" title="disabled">'+testsuite.disabled+'</span>';
    }
    if (testsuite.errors > 0) {
      badges += '<span class="badge rounded-pill float-right text-bg-danger" title="errors">'+testsuite.errors+'</span>';
    }
    
    var ok = testsuite.total - testsuite.errors - testsuite.disabled;
    if (ok > 0) {
      badges += '<span class="badge rounded-pill float-right text-bg-success" title="success">'+ok+'</span>';
    }
    
    var r = '<li class="nav-item">';
    r += '<a class="nav-link" href="#" id="'+idx+'">';
    r += '  <span data-feather="file-text" class="align-text-bottom"></span>';
    r += '  '+testsuite.name+' <small>'+badges+'</small>';
    r += '</a>';
    r += '</li>';
    return r;
  }

  function middletestsuite(idx, testsuite) {
    
    $('#testcases').html('');
    if (testsuite.testcases) {
      var testcases = "";
      for (var i = 0; i < testsuite.testcases.length; i++) {
        var testcase = testsuite.testcases[i];
        var id = i;

        var status = "success";
        var body = "";

        if (testcase.errors && testcase.errors.length > 0) {
          status = "danger";
          for (var j = 0; j < testcase.errors.length; j++) {
            body += '<div class="container">Error n°'+j+' <pre>'+testcase.errors[j].value+'</pre></div>';
          }
        }

        if (testcase.failures && testcase.failures.length > 0) {
          status = "danger";
          for (var j = 0; j < testcase.failures.length; j++) {
            body += '<div class="container">Error n°'+j+' <pre>'+testcase.failures[j].value+'</pre></div>';
          }
        }

        if (testcase.systemout && testcase.systemout.value != "") {
          body += '<div class="container">Systemout: <pre>'+testcase.systemout.value+'</pre></div>';
        }
        if (testcase.systemerr && testcase.systemerr.value != "") {
          body += '<div class="container">Systemerr: <pre>'+testcase.systemerr.value+'</pre></div>';
        }

        var r = '<div class="accordion-item">';
        r += '<h2 class="accordion-header" id="testcase-'+id+'-heading">';
        r += '  <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#testcase-'+id+'-collapseOne" aria-expanded="true" aria-controls="testcase-'+id+'-collapseOne">';
        r += '<div class="alert alert-'+status+'" role="alert">';
        r += ''+testcase.name;
        r += '<span class="position-absolute top-0 start-100 translate-middle badge rounded-pill bg-secondary">'+parseFloat(testcase.time).toFixed(2)+'s</span>';
        r += '</div>';
        r += '  </button>';
        r += '</h2>';
        r += '<div id="testcase-'+id+'-collapseOne" class="accordion-collapse collapse show" aria-labelledby="testcase-'+id+'-heading">';
        r += '  <div class="accordion-body">';
        r += '    '+body;
        r += '  </div>';
        r += '</div>';
        r += '</div>';

        testcases += r;
      }
      $('#testcases').html(testcases);
    }
  }

  function dis(id, value, sing, plu) {
    $(id).hide()
    if (value == 1) {
      $(id).html(value + " " + sing);
      $(id).show()
    } else if (value > 1) {
        $(id).html(value + " " + plu);
        $(id).show()
    }
  }
  </script>  
  </body>
</html>
`

type TestsHTML struct {
	Tests     Tests  `json:"tests"`
	JSONValue string `json:"jsonValue"`
}

func outputHTML(testsResult *Tests) ([]byte, error) {
	var buf bytes.Buffer

	testJSON, err := json.Marshal(testsResult)
	if err != nil {
		return nil, errors.Wrap(err, "unable to make json value")
	}

	testsHTML := TestsHTML{
		Tests:     *testsResult,
		JSONValue: string(testJSON),
	}
	tmpl := template.Must(template.New("reportHTML").Parse(templateHTML))
	if err := tmpl.Execute(&buf, testsHTML); err != nil {
		return nil, errors.Wrap(err, "unable to make template")
	}
	return buf.Bytes(), nil
}
