Feature: Document Types API

  @atomic
  Scenario: Get all document types
    Given the API is available
    When I call API "GetDocumentTypes"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new document type
    Given the API is available
    And I have the following request body:
    """
    {
      "code": "LEGAL",
      "name": "Legal Document",
      "description": "Documents related to legal matters."
    }
    """
    When I call API "CreateDocumentType"
    Then response status should be 200

  @atomic
  Scenario: Get document type by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetDocumentTypeByID"
    Then response status should be 200

  @atomic
  Scenario: Update a document type
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "code": "LEGAL-V2",
      "name": "Legal Document V2",
      "description": "Updated documents related to legal matters."
    }
    """
    When I call API "UpdateDocumentType"
    Then response status should be 200

  @atomic
  Scenario: Delete a document type
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteDocumentType"
    Then response status should be 200
