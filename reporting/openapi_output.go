package reporting

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

var KnownHTTPMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
	"CONNECT": true,
}

// ----------------------------------------------------------------------
// OpenAPI & PathItem Structs
// ----------------------------------------------------------------------

type OpenAPI struct {
	Openapi    string                 `json:"openapi"`
	Info       Info                   `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]*PathItem   `json:"paths"`
	Components map[string]interface{} `json:"components,omitempty"`
	Tags       []Tag                  `json:"tags,omitempty"`
}

type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Trace   *Operation `json:"trace,omitempty"`
}

type Operation struct {
	Tags        []string `json:"tags,omitempty"`
	OperationID string   `json:"operationId,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Server struct {
	URL string `json:"url"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ----------------------------------------------------------------------
// JUnit Data Structures
// ----------------------------------------------------------------------

type Testcase struct {
	Name      string    `xml:"name,attr"`
	Classname string    `xml:"classname,attr"`
	Failures  []Failure `xml:"failure"`
	Error     *Failure  `xml:"error"`
	Skipped   *Skipped  `xml:"skipped"`
}

type Failure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type Skipped struct {
	Message string `xml:"message,attr"`
}

type Testsuite struct {
	Name      string     `xml:"name,attr"`
	Testcases []Testcase `xml:"testcase"`
}

type TestSuites struct {
	Testsuites []Testsuite `xml:"testsuite"`
}

type FileEntry struct {
	Path  string
	Entry os.DirEntry
}

// ----------------------------------------------------------------------
// Coverage Data Structures
// ----------------------------------------------------------------------

// EndpointCoverage holds coverage info for a single endpoint
type EndpointCoverage struct {
	Method       string
	Path         string
	TotalTests   int
	PassedTests  int
	FailedTests  int
	CoveragePct  float64
	CoverageType string // e.g., "full", "partial", "empty"
	Conditions   []ConditionCoverage
	Tags         []string
}

type ConditionCoverage struct {
	Name   string
	Passed bool
	Detail string
}

type ReportData struct {
	FullCoverage    float64
	PartialCoverage float64
	EmptyCoverage   float64
	TotalEndpoints  int
	CoverageData    []EndpointCoverage
	GroupedCoverage map[string]GroupCoverage
}

type GroupCoverage struct {
	Coverages   []EndpointCoverage
	CoveragePct float64
}

// ----------------------------------------------------------------------
// File Loading: OpenAPI & JUnit
// ----------------------------------------------------------------------

// LoadOpenAPISpec loads an OpenAPI JSON file into the typed `OpenAPI` struct
func LoadOpenAPISpec(filename string) (*OpenAPI, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var openAPI OpenAPI
	if err := json.Unmarshal(data, &openAPI); err != nil {
		return nil, err
	}
	return &openAPI, nil
}

// LoadJUnitXML loads a JUnit XML file into `TestSuites` struct
func LoadJUnitXML(filename string) (*TestSuites, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var testsuites TestSuites
	if err := xml.Unmarshal(data, &testsuites); err != nil {
		return nil, err
	}
	return &testsuites, nil
}

// ----------------------------------------------------------------------
// Coverage Calculation
// ----------------------------------------------------------------------

// CalculateCoverage enumerates all endpoints from the typed `OpenAPI.Paths`
// and matches them with your test suites using `ExtractHttpEndpoint`.
func CalculateCoverage(openAPI *OpenAPI, testSuites *TestSuites) []EndpointCoverage {
	endpoints := getAllEndpointsWithTags(openAPI)

	coverageReport := make([]EndpointCoverage, len(endpoints))
	copy(coverageReport, endpoints)

	endpointMap := make(map[string]*EndpointCoverage)
	for i := range coverageReport {
		key := fmt.Sprintf("%s %s", coverageReport[i].Method, coverageReport[i].Path)
		endpointMap[key] = &coverageReport[i]
	}

	for _, ts := range testSuites.Testsuites {
		method, endpoint := ExtractHttpEndpoint(ts.Name)
		if method == "" || endpoint == "" {
			continue
		}

		key := fmt.Sprintf("%s %s", method, endpoint)
		ec, found := endpointMap[key]
		if !found {
			continue
		}

		for _, tc := range ts.Testcases {
			ec.TotalTests++
			passed := (len(tc.Failures) == 0 && tc.Error == nil && tc.Skipped == nil)
			if passed {
				ec.PassedTests++
			} else {
				ec.FailedTests++
			}

			cond := ConditionCoverage{
				Name:   tc.Name,
				Passed: passed,
				Detail: fmt.Sprintf("Class: %s", tc.Classname),
			}

			if !passed {
				if len(tc.Failures) > 0 {
					cond.Detail += " | Failure: " + tc.Failures[0].Message
				}
				if tc.Error != nil {
					cond.Detail += " | Error: " + tc.Error.Message
				}
				if tc.Skipped != nil {
					cond.Detail += " | Skipped: " + tc.Skipped.Message
				}
			}
			ec.Conditions = append(ec.Conditions, cond)
		}

		if ec.TotalTests > 0 {
			ec.CoveragePct = float64(ec.PassedTests) / float64(ec.TotalTests) * 100.0
			switch {
			case ec.CoveragePct == 100:
				ec.CoverageType = "full"
			case ec.CoveragePct > 0:
				ec.CoverageType = "partial"
			default:
				ec.CoverageType = "empty"
			}
		} else {
			ec.CoverageType = "empty"
		}
	}

	return coverageReport
}

func getAllEndpointsWithTags(openAPI *OpenAPI) []EndpointCoverage {
	var endpoints []EndpointCoverage
	for path, item := range openAPI.Paths {

		// GET
		if item.Get != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "GET",
				Path:         path,
				Tags:         item.Get.Tags,
				CoverageType: "empty", // default
			})
		}
		// POST
		if item.Post != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "POST",
				Path:         path,
				Tags:         item.Post.Tags,
				CoverageType: "empty",
			})
		}
		// PUT
		if item.Put != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "PUT",
				Path:         path,
				Tags:         item.Put.Tags,
				CoverageType: "empty",
			})
		}
		// PATCH
		if item.Patch != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "PATCH",
				Path:         path,
				Tags:         item.Patch.Tags,
				CoverageType: "empty",
			})
		}
		// DELETE
		if item.Delete != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "DELETE",
				Path:         path,
				Tags:         item.Delete.Tags,
				CoverageType: "empty",
			})
		}
		// HEAD
		if item.Head != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "HEAD",
				Path:         path,
				Tags:         item.Head.Tags,
				CoverageType: "empty",
			})
		}
		// OPTIONS
		if item.Options != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "OPTIONS",
				Path:         path,
				Tags:         item.Options.Tags,
				CoverageType: "empty",
			})
		}
		// TRACE
		if item.Trace != nil {
			endpoints = append(endpoints, EndpointCoverage{
				Method:       "TRACE",
				Path:         path,
				Tags:         item.Trace.Tags,
				CoverageType: "empty",
			})
		}
	}
	return endpoints
}

// ----------------------------------------------------------------------
// Extracting Method/Endpoint from Testsuite Name
// ----------------------------------------------------------------------

// ExtractHttpEndpoint tries to parse testsuite name for "TestSuite for GET /user/{username}"
func ExtractHttpEndpoint(name string) (string, string) {
	parts := strings.Fields(name)
	// e.g. "TestSuite for GET /some/endpoint"
	if len(parts) >= 5 &&
		strings.EqualFold(parts[0], "testsuite") &&
		strings.EqualFold(parts[1], "for") {

		method := strings.ToUpper(parts[2])
		if !KnownHTTPMethods[method] {
			return "", ""
		}
		// endpoint is the next chunk, e.g. "/some/endpoint"
		if len(parts) > 5 {
			// optional check for extra tokens
			return "", ""
		}
		endpoint := strings.Join(parts[3:4], " ")
		return method, endpoint
	}
	return "", ""
}

// GroupCoverageByTag groups endpoints by their first listed tag.
func GroupCoverageByTag(coverages []EndpointCoverage) map[string]GroupCoverage {
	tmp := make(map[string][]EndpointCoverage)
	for _, cov := range coverages {
		if len(cov.Tags) > 0 {
			tag := cov.Tags[0]
			tmp[tag] = append(tmp[tag], cov)
		} else {
			tmp["untagged"] = append(tmp["untagged"], cov)
		}
	}

	result := make(map[string]GroupCoverage)
	for tag, endpoints := range tmp {
		total := len(endpoints)
		if total == 0 {
			result[tag] = GroupCoverage{
				Coverages:   endpoints,
				CoveragePct: 0,
			}
			continue
		}

		covered := 0
		for _, ec := range endpoints {
			if ec.CoverageType != "empty" {
				covered++
			}
		}

		coveragePct := (float64(covered) / float64(total)) * 100.0
		result[tag] = GroupCoverage{
			Coverages:   endpoints,
			CoveragePct: coveragePct,
		}
	}

	return result
}
