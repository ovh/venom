package venom

import (
	"encoding/json"
	"encoding/xml"
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

// OpenAPI represents the OpenAPI specification
type OpenAPI struct {
	Paths map[string]interface{} `json:"paths"`
}

type Testsuite struct {
	Name string `xml:"name,attr"`
}

type TestSuites struct {
	TestSuites []Testsuite `xml:"testsuite"`
}

type FileEntry struct {
	Path  string
	Entry os.DirEntry
}

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

func GetAllEndpoints(openAPI *OpenAPI) map[string][]string {
	endpoints := make(map[string][]string)

	for path, methods := range openAPI.Paths {
		methodList := make([]string, 0)
		for method := range methods.(map[string]interface{}) {
			methodList = append(methodList, strings.ToUpper(method))
		}
		endpoints[path] = methodList
	}

	return endpoints
}

func IsHTTPMethod(method string) bool {
	return KnownHTTPMethods[method]
}

func ExtractHttpEndpoint(name string) (string, string) {
	parts := strings.Fields(name)
	// check prefix "TestSuite for"
	if len(parts) >= 5 && strings.EqualFold(parts[0], "testsuite") && strings.EqualFold(parts[1], "for") {
		httpMethod := strings.ToUpper(parts[2])
		if IsHTTPMethod(httpMethod) {
			if len(parts) > 5 {
				//TODO: Need to make name convention contain 1 api or use new description available for testsuites in venom
				return "", ""
			}
			endpoint := strings.Join(parts[3:4], " ")
			return httpMethod, endpoint
		}
	}
	return "", ""
}
