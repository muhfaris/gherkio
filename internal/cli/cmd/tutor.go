package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var tutorCmd = &cobra.Command{
	Use:   "tutor",
	Short: "An interactive tutorial for gherkio",
	Long:  `Learn gherkio by following this interactive, multi-level tutorial.`,
	Run: func(cmd *cobra.Command, args []string) {
		startInteractiveTutor()
	},
}

// Main entry point for the tutorial
func startInteractiveTutor() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the gherkio interactive tutorial!")
	time.Sleep(1 * time.Second)

	for {
		fmt.Println("\nChoose a tutorial level:")
		fmt.Println("  1. Simple: Basic API testing")
		fmt.Println("  2. Advanced: Chaining requests")
		fmt.Println("  3. Complex: Dynamic data & custom headers")
		fmt.Println("  4. Professional: Using fixtures")
		fmt.Println("  5. Expert: Reusable flows")
		fmt.Println("  6. Exit")
		fmt.Print("> ")

		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			runSimpleTutorial(reader)
		case "2":
			runAdvancedTutorial(reader)
		case "3":
			runComplexTutorial(reader)
		case "4":
			runProfessionalTutorial(reader)
		case "5":
			runExpertTutorial(reader)
		case "6":
			fmt.Println("Exiting tutorial. Happy testing!")
			return
		default:
			fmt.Println("Invalid choice. Please enter a number from 1 to 6.")
		}
		fmt.Println("\n-------------------------------------------------")
	}
}

// Simple Tutorial: Basic GET request
func runSimpleTutorial(reader *bufio.Reader) {
	fmt.Println("\n--- Simple Tutorial: Basic GET Request ---")
	fmt.Println("This tutorial shows how to make a simple GET request and verify the response.")
	fmt.Println("\nFirst, we need a .feature file to define the test. Here is `examples/simple_get.feature`:")
	fmt.Println("```gherkin")
	fmt.Println(`Feature: Simple API Test
  Scenario: Get a user
    When I call API "httpbin.get"
    Then response status should be 200`)
	fmt.Println("```")
	fmt.Println("\nTo make this work, we need a catalog file (`gherkio.yml`) to define the `httpbin.get` API call:")
	fmt.Println("```yaml")
	fmt.Println(`apis:
  httpbin.get:
    method: GET
    url: https://httpbin.org/get`)
	fmt.Println("```")
	fmt.Println("\nNow, let's run this test. Type the following command and press Enter:")

	promptAndValidate(reader, "gherkio run examples/simple_get.feature", simulatedSimpleGetOutput)

	fmt.Println("\nGreat! You've successfully run a simple test.")
	fmt.Println("You defined an API call in a catalog and used Gherkin to execute it and verify the status.")
}

// Advanced Tutorial: Chaining Requests
func runAdvancedTutorial(reader *bufio.Reader) {
	fmt.Println("\n--- Advanced Tutorial: Chaining Requests ---")
	fmt.Println("This tutorial demonstrates how to chain requests, a common pattern in API testing.")
	fmt.Println("We will create a resource with a POST request, save a value from its response, and then use that value in a subsequent GET request.")
	fmt.Println("\nHere is `examples/advanced_chaining.feature`:")
	fmt.Println("```gherkin")
	fmt.Println(`Feature: Chained API Requests
  Scenario: Create a user then get the user
    When I call API "httpbin.post" with body:
      """
      {"name": "Jules", "job": "Engineer"}
      """
    Then response status should be 200
    And save "json.data" as "userJson"

  Scenario: Get the created user data
    When I set query params:
      | user | {{userJson}} |
    And I call API "httpbin.get"
    Then response status should be 200
    And json "args.user" should equal "{\"name\":\"Jules\",\"job\":\"Engineer\"}"`)
	fmt.Println("```")
	fmt.Println("\nNotice how we save the `json.data` object as `userJson` and then use `{{userJson}}` as a query parameter in the next request.")
	fmt.Println("Let's run it. Type the following command:")

	promptAndValidate(reader, "gherkio run examples/advanced_chaining.feature", simulatedAdvancedChainingOutput)

	fmt.Println("\nExcellent! You've chained a POST and a GET request.")
	fmt.Println("This technique is crucial for testing workflows that span multiple API calls.")
}

// Complex Tutorial: Dynamic Data and Custom Headers
func runComplexTutorial(reader *bufio.Reader) {
	fmt.Println("\n--- Complex Tutorial: Dynamic Data & Headers ---")
	fmt.Println("This tutorial covers advanced features like generating dynamic data and adding custom headers.")
	fmt.Println("We'll use a function template to generate a random email and send a custom API key in the headers.")
	fmt.Println("\nHere is `examples/complex_test.feature`:")
	fmt.Println("```gherkin")
	fmt.Println(`Feature: Complex API Test
  Scenario: Create a user with dynamic data and custom headers
    Given I set headers:
      | X-API-Key | my-secret-key |
    When I call API "httpbin.post" with body:
      """
      {"email": "{{fnRandomEmail}}", "role": "user"}
      """
    Then response status should be 200
    And json "headers.X-Api-Key" should equal "my-secret-key"
    And json "json.email" should match "@"`)
	fmt.Println("```")
	fmt.Println("\nNote the use of `{{fnRandomEmail}}` and the `I set headers` step.")
	fmt.Println("Let's run this complex test. Type the command:")

	promptAndValidate(reader, "gherkio run examples/complex_test.feature", simulatedComplexOutput)

	fmt.Println("\nFantastic! You've mastered dynamic data and custom headers.")
	fmt.Println("Function templates are powerful tools for creating realistic and varied test data.")
}

// Professional Tutorial: Using Fixtures
func runProfessionalTutorial(reader *bufio.Reader) {
	fmt.Println("\n--- Professional Tutorial: Using Fixtures ---")
	fmt.Println("This tutorial teaches you how to keep your test data separate from your test logic using fixture files.")
	fmt.Println("Fixtures are great for managing large or complex JSON payloads.")
	fmt.Println("\nFirst, create a fixture file named `gherkio/fixtures/user.json`:")
	fmt.Println("```json")
	fmt.Println(`{
  "body": {
    "name": "Jules from Fixture",
    "job": "Lead Engineer"
  }
}`)
	fmt.Println("```")
	fmt.Println("\nNow, use this fixture in your Gherkin file, `examples/fixture_test.feature`:")
	fmt.Println("```gherkin")
	fmt.Println(`Feature: API Test with Fixture
  Scenario: Create a user from a fixture file
    When I call API "httpbin.post" using fixture "user.json"
    Then response status should be 200
    And json "json.name" should equal "Jules from Fixture"`)
	fmt.Println("```")
	fmt.Println("\nLet's run it. Type the command:")

	promptAndValidate(reader, "gherkio run examples/fixture_test.feature", simulatedFixtureOutput)

	fmt.Println("\nWell done! You have cleanly separated your test data using a fixture.")
	fmt.Println("This makes both your Gherkin files and your data payloads easier to manage and reuse.")
}

// Expert Tutorial: Reusable Flows
func runExpertTutorial(reader *bufio.Reader) {
	fmt.Println("\n--- Expert Tutorial: Reusable Flows ---")
	fmt.Println("This tutorial introduces flows, which allow you to define reusable sequences of API calls.")
	fmt.Println("Flows are perfect for common actions like authentication or complex setup procedures.")
	fmt.Println("\nDefine a flow in your `gherkio.yml` file:")
	fmt.Println("```yaml")
	fmt.Println(`flows:
  authenticate:
    - call: httpbin.post
      body: '{"user": "admin", "pass": "secret"}'
      save:
        json.headers.Authorization: bearer_token`)
	fmt.Println("```")
	fmt.Println("\nNow, you can execute this entire sequence with a single step in `examples/flow_test.feature`:")
	fmt.Println("```gherkin")
	fmt.Println(`Feature: API Test with a Flow
  Scenario: Authenticate and then get a protected resource
    When I run flow "authenticate"
    And I set headers:
      | Authorization | {{bearer_token}} |
    And I call API "httpbin.get"
    Then response status should be 200`)
	fmt.Println("```")
	fmt.Println("\nLet's run it. Type the command:")

	promptAndValidate(reader, "gherkio run examples/flow_test.feature", simulatedFlowOutput)

	fmt.Println("\nCongratulations! You have successfully used a flow to simplify your test.")
	fmt.Println("Flows are a key feature for writing clean, maintainable, and powerful API tests.")
}


func promptAndValidate(reader *bufio.Reader, expectedCmd string, outputToShow func()) {
	// This function will be used by all tutorial levels
	for {
		fmt.Printf("\n> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.EqualFold(input, expectedCmd) {
			fmt.Println("\nCorrect! Here is a simulation of the output:")
			time.Sleep(500 * time.Millisecond)
			fmt.Println("...")
			time.Sleep(1 * time.Second)
			outputToShow()
			break
		} else {
			fmt.Println("Not quite. Please type the command exactly as shown and press Enter.")
		}
	}
}

// Simulation functions will be added as each tutorial level is implemented
func simulatedSimpleGetOutput() {
	fmt.Print(`
Feature: Simple API Test

  Scenario: Get a user
    When I call API "httpbin.get"
    Then response status should be 200

1 scenario (1 passed)
2 steps (2 passed)
`)
}

func simulatedAdvancedChainingOutput() {
	fmt.Print(`
Feature: Chained API Requests

  Scenario: Create a user then get the user
    ...
    ✔ Then response status should be 200
    ✔ And save "json.data" as "userJson"

  Scenario: Get the created user data
    ...
    ✔ Then response status should be 200
    ✔ And json "args.user" should equal "{\"name\":\"Jules\",\"job\":\"Engineer\"}"

2 scenarios (2 passed)
7 steps (7 passed)
`)
}

func simulatedComplexOutput() {
	fmt.Print(`
Feature: Complex API Test

  Scenario: Create a user with dynamic data and custom headers
    ...
    ✔ Then response status should be 200
    ✔ And json "headers.X-Api-Key" should equal "my-secret-key"
    ✔ And json "json.email" should match "@"

1 scenario (1 passed)
6 steps (6 passed)
`)
}

func simulatedFixtureOutput() {
	fmt.Print(`
Feature: API Test with Fixture

  Scenario: Create a user from a fixture file
    When I call API "httpbin.post" using fixture "user.json"
    Then response status should be 200
    And json "json.name" should equal "Jules from Fixture"

1 scenario (1 passed)
3 steps (3 passed)
`)
}

func simulatedFlowOutput() {
	fmt.Print(`
Feature: API Test with a Flow

  Scenario: Authenticate and then get a protected resource
    When I run flow "authenticate"
    And I set headers:
      | Authorization | Bearer my-secret-token |
    And I call API "httpbin.get"
    Then response status should be 200

1 scenario (1 passed)
4 steps (4 passed)
`)
}

func init() {
	rootCmd.AddCommand(tutorCmd)
}
