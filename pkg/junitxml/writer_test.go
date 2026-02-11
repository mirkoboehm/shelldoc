package junitxml

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func openTmpFile() (*os.File, error) {
	file, err := os.CreateTemp("", "write_test-*.xml")
	if err != nil {
		return nil, fmt.Errorf("unable to open temporary output file: %v", err)
	}
	return file, nil
}

func removeTmpFile(filepath string) {
	const variable = "SHELLDOC_TEST_KEEP_TEMPORARY_FILES"
	if _, isSet := os.LookupEnv(variable); isSet {
		fmt.Printf("%s is set, not removing temporary file %s\n", variable, filepath)
		return
	}
	if err := os.Remove(filepath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to remove temporary file at %s: %d", filepath, err)
	}
}

func validateXMLFile(filepath string) error {
	cmd := exec.Command("xmllint", "--noout", "--schema", "jenkins-junit.xsd", filepath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("XML validation finished with error: %v", err)
	}
	return nil
}

func TestMinimalDocument(t *testing.T) {
	// Write a minimal XML file with an empty testsuites section.
	testsuites := JUnitTestSuites{}

	file, err := openTmpFile()
	require.NoError(t, err, "Unable to open file for temporary XML document")
	defer removeTmpFile(file.Name())

	err = testsuites.Write(file)
	require.NoError(t, err, "Unable to write temporary XML document")
	// Verify it is schema compliant.
	require.NoError(t, validateXMLFile(file.Name()), "XML document fails to validate")
}

func TestOneTestSuite(t *testing.T) {
	// Write a minimal XML file with an empty testsuites section.
	testsuites := JUnitTestSuites{}
	ts := JUnitTestSuite{
		Tests:      1,
		Failures:   1,
		Time:       FormatTime(1234000000),
		Name:       "Test-TestSuite",
		Properties: []JUnitProperty{},
		TestCases:  []JUnitTestCase{},
	}
	ts.AddProperty("go.version", runtime.Version())

	testCase := JUnitTestCase{
		Classname: "README.md",
		Name:      "ls -l",
		Time:      FormatTime(51345000),
		Failure: &JUnitFailure{
			Message:  "Failed",
			Type:     "mismatch",
			Contents: "(the test output)",
		},
	}
	ts.TestCases = append(ts.TestCases, testCase)
	testsuites.Suites = append(testsuites.Suites, ts)

	// The rest should be data/table driven...:
	file, err := openTmpFile()
	require.NoError(t, err, "Unable to open file for temporary XML document")
	defer removeTmpFile(file.Name())

	err = testsuites.Write(file)
	require.NoError(t, err, "Unable to write temporary XML document")
	// Verify it is schema compliant.
	require.NoError(t, validateXMLFile(file.Name()), "XML document fails to validate")
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0.000"},
		{time.Second, "1.000"},
		{1500 * time.Millisecond, "1.500"},
		{123 * time.Millisecond, "0.123"},
	}
	for _, tc := range tests {
		result := FormatTime(tc.duration)
		require.Equal(t, tc.expected, result)
	}
}

func TestFormatBenchmarkTime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0.000000000"},
		{time.Second, "1.000000000"},
		{123456789 * time.Nanosecond, "0.123456789"},
	}
	for _, tc := range tests {
		result := FormatBenchmarkTime(tc.duration)
		require.Equal(t, tc.expected, result)
	}
}

func TestRegisterTestCase(t *testing.T) {
	suite := &JUnitTestSuite{Name: "test-suite"}

	// Register a successful test case
	tc1 := JUnitTestCase{Name: "test1", Classname: "class1"}
	err := suite.RegisterTestCase(tc1)
	require.NoError(t, err)
	require.Equal(t, 1, suite.Tests)
	require.Equal(t, 1, suite.TestCount())
	require.Equal(t, 0, suite.Failures)
	require.Equal(t, 0, suite.Errors)

	// Register a failed test case
	tc2 := JUnitTestCase{Name: "test2", Classname: "class1"}
	tc2.RegisterFailure("FAILURE", "test failed", "details")
	err = suite.RegisterTestCase(tc2)
	require.NoError(t, err)
	require.Equal(t, 2, suite.Tests)
	require.Equal(t, 1, suite.Failures)

	// Register a test case with error
	tc3 := JUnitTestCase{Name: "test3", Classname: "class1"}
	tc3.RegisterError("ERROR", "execution error", "details")
	err = suite.RegisterTestCase(tc3)
	require.NoError(t, err)
	require.Equal(t, 3, suite.Tests)
	require.Equal(t, 1, suite.Errors)
}

func TestSuiteCounters(t *testing.T) {
	suite := &JUnitTestSuite{Name: "test-suite"}

	// Add a successful test
	suite.RegisterTestCase(JUnitTestCase{Name: "success1"})

	// Add a failed test
	failedTC := JUnitTestCase{Name: "failed1"}
	failedTC.RegisterFailure("FAILURE", "msg", "contents")
	suite.RegisterTestCase(failedTC)

	// Add an error test
	errorTC := JUnitTestCase{Name: "error1"}
	errorTC.RegisterError("ERROR", "msg", "contents")
	suite.RegisterTestCase(errorTC)

	// Add another successful test
	suite.RegisterTestCase(JUnitTestCase{Name: "success2"})

	require.Equal(t, 4, suite.TestCount())
	require.Equal(t, 2, suite.SuccessCount())
	require.Equal(t, 1, suite.FailureCount())
	require.Equal(t, 1, suite.ErrorCount())
}

func TestRegisterFailure(t *testing.T) {
	tc := &JUnitTestCase{Name: "test"}
	require.Nil(t, tc.Failure)

	tc.RegisterFailure("FAILURE", "test failed", "expected X got Y")

	require.NotNil(t, tc.Failure)
	require.Equal(t, "FAILURE", tc.Failure.Type)
	require.Equal(t, "test failed", tc.Failure.Message)
	require.Equal(t, "expected X got Y", tc.Failure.Contents)
}

func TestRegisterError(t *testing.T) {
	tc := &JUnitTestCase{Name: "test"}
	require.Nil(t, tc.Error)

	tc.RegisterError("ERROR", "execution failed", "stack trace")

	require.NotNil(t, tc.Error)
	require.Equal(t, "ERROR", tc.Error.Type)
	require.Equal(t, "execution failed", tc.Error.Message)
	require.Equal(t, "stack trace", tc.Error.Contents)
}

func TestAddProperty(t *testing.T) {
	suite := &JUnitTestSuite{Name: "test-suite"}
	require.Empty(t, suite.Properties)

	suite.AddProperty("key1", "value1")
	suite.AddProperty("key2", "value2")

	require.Len(t, suite.Properties, 2)
	require.Equal(t, "key1", suite.Properties[0].Name)
	require.Equal(t, "value1", suite.Properties[0].Value)
	require.Equal(t, "key2", suite.Properties[1].Name)
	require.Equal(t, "value2", suite.Properties[1].Value)
}

func TestRegisterElapsedTime(t *testing.T) {
	var timeStr string
	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	RegisterElapsedTime(start, &timeStr)

	require.NotEmpty(t, timeStr)
	// The time should be at least 0.010 seconds (10ms)
	require.Contains(t, timeStr, ".")
}
