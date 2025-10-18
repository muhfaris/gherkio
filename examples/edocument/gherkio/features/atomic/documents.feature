Feature: Documents API

  @atomic
  Scenario: Get all documents
    Given the API is available
    When I call API "GetDocuments"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new document
    Given the API is available
    And I have the following request body:
    """
    {
      "name": "New Document",
      "code": "DOC-001",
      "description": "This is a test document."
    }
    """
    When I call API "CreateDocument"
    Then response status should be 200

  @atomic
  Scenario: Get document by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetDocumentByID"
    Then response status should be 200

  @atomic
  Scenario: Update a document
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "name": "Updated Document",
      "code": "DOC-001-rev1",
      "description": "This is an updated test document."
    }
    """
    When I call API "UpdateDocument"
    Then response status should be 200

  @atomic
  Scenario: Delete a document
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteDocument"
    Then response status should be 200
