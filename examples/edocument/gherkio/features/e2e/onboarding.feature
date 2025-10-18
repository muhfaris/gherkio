@e2e
Feature: End-to-End Scenarios

  Scenario: Admin Authentication and Verification
    # 1. Login as admin and get a token
    When I run flow "authenticate-admin"

    # 2. Use the captured token to access a protected endpoint
    # Note: The bearer auth is configured in apis/core.yaml to automatically use 'access_token' from the store.
    # We will need to add a step in our flow to move the captured token to the expected key.
    Given I have a variable "access_token" from the store at "admin_token"
    When I call API "MyInfo"
    Then response status should be 200
    And the response body path "email" should not be empty
