Feature: Companies API

  @atomic
  Scenario: Get all companies
    Given the API is available
    When I call API "GetCompanies"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new company
    Given the API is available
    And I have the following request body:
    """
    {
      "name": "New Company",
      "description": "This is a test company.",
      "isActive": true
    }
    """
    When I call API "CreateCompany"
    Then response status should be 200

  @atomic
  Scenario: Get company by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetCompanyByID"
    Then response status should be 200

  @atomic
  Scenario: Update a company
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "name": "Updated Company",
      "description": "This is an updated test company.",
      "isActive": false
    }
    """
    When I call API "UpdateCompany"
    Then response status should be 200

  @atomic
  Scenario: Delete a company
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteCompany"
    Then response status should be 200
