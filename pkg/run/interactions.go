package run

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/mirkoboehm/shelldoc/pkg/junitxml"
	"github.com/mirkoboehm/shelldoc/pkg/shell"
	"github.com/mirkoboehm/shelldoc/pkg/tokenizer"
	"github.com/mirkoboehm/shelldoc/pkg/version"
)

const (
	returnSuccess = iota // the test succeeded
	returnFailure        // the test failed (a problemn with the test)
	returnError          // there was an error executing the test (a problem with shelldoc)
)

func result(code int) string {
	switch code {
	case returnFailure:
		return "FAILURE"
	case returnError:
		return "ERROR"
	default:
		return "SUCCESS"
	}
}

func (context *Context) performInteractions(inputfile string) (*junitxml.JUnitTestSuite, error) {
	// the test suite object for this file
	suite := &junitxml.JUnitTestSuite{Name: inputfile}
	suite.AddProperty("shelldoc-version", version.Version())
	defer junitxml.RegisterElapsedTime(time.Now(), &suite.Time)
	// read input data
	data, err := ReadInput([]string{inputfile})
	if err != nil {
		return nil, fmt.Errorf("unable to read input data: %v", err)
	}
	// run the input through the tokenizer
	visitor := tokenizer.NewInteractionVisitor()
	tokenizer.Tokenize(data, visitor)

	// dry-run mode: just list the commands without executing them
	if context.DryRun {
		fmt.Printf("SHELLDOC: dry-run \"%s\" ...\n", inputfile)
		magnitude := int(math.Log10(float64(len(visitor.Interactions)))) + 1
		counterFormat := fmt.Sprintf("%%%ds", magnitude+2)
		opener := fmt.Sprintf(" CMD %s: %%s\n", counterFormat)
		for index, interaction := range visitor.Interactions {
			fmt.Printf(opener, fmt.Sprintf("(%d)", index+1), interaction.Cmd)
		}
		fmt.Printf("Found %d commands (dry-run, not executed)\n", len(visitor.Interactions))
		return suite, nil
	}

	// detect shell
	shellpath, err := shell.DetectShell(context.ShellName)
	if err != nil {
		return nil, err
	}
	// start a background shell, it will run until the function ends
	currentShell, err := shell.StartShell(shellpath)
	if err != nil {
		return nil, fmt.Errorf("unable to start shell: %v", err)
	}
	defer currentShell.Exit()

	// execute the interactions and verify the results:
	fmt.Printf("SHELLDOC: doc-testing \"%s\" ...\n", inputfile)
	// construct the opener and closer format strings, since they depend on verbose mode
	magnitude := int(math.Log10(float64(len(visitor.Interactions)))) + 1
	openerLineEnding := "  : "
	resultString := " "
	if context.Verbose {
		openerLineEnding = "\n"
		resultString = " <-- "
	}
	counterFormat := fmt.Sprintf("%%%ds", magnitude+2)
	opener := fmt.Sprintf(" CMD %s: %%s%s", counterFormat, openerLineEnding)
	closer := fmt.Sprintf("%s%%s\n", resultString)

	for index, interaction := range visitor.Interactions {
		fmt.Printf(opener, fmt.Sprintf("(%d)", index+1), interaction.Describe())
		if context.Verbose {
			fmt.Printf(" --> %s\n", interaction.Cmd)
		}
		testcase, err := context.performTestCase(interaction, &currentShell)
		testcase.Classname = inputfile // testcase is always returned, even if err is not nil
		if context.ReplaceDots {
			testcase.Classname = strings.ReplaceAll(inputfile, ".", "●")
		}
		if err != nil {
			fmt.Printf(" --  ERROR: %v", err)
			context.RegisterReturnCode(returnError)
			testcase.RegisterError(result(returnError), interaction.Result(), err.Error())
		}
		fmt.Printf(closer, interaction.Result())
		if interaction.HasFailure() {
			context.RegisterReturnCode(returnFailure)
			testcase.RegisterFailure(result(returnFailure), interaction.Result(), interaction.DescribeFull())
		}
		if err := suite.RegisterTestCase(*testcase); err != nil {
			return nil, fmt.Errorf("failed to register test case: %w", err)
		}
		// Abort on timeout - shell state is undefined after a timeout
		if err == shell.ErrTimeout {
			log.Printf("Aborting test run due to timeout.")
			break
		}
		if interaction.HasFailure() && context.FailureStops {
			log.Printf("Stop requested after first failed test.")
			break
		}
	}
	fmt.Printf("%s: %d tests - %d successful, %d failures, %d errors\n", result(context.ReturnCode()), suite.TestCount(),
		suite.SuccessCount(), suite.FailureCount(), suite.ErrorCount())
	return suite, nil
}

func (context *Context) performTestCase(interaction *tokenizer.Interaction, sh *shell.Shell) (*junitxml.JUnitTestCase, error) {
	testcase := &junitxml.JUnitTestCase{
		Name: interaction.Cmd,
	}
	defer junitxml.RegisterElapsedTime(time.Now(), &testcase.Time)
	return testcase, interaction.Execute(sh, context.Timeout)
}
