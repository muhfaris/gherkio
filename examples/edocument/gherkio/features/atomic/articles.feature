Feature: Articles API

  @atomic
  Scenario: Get all articles
    Given the API is available
    When I call API "GetArticles"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new article
    Given the API is available
    And I have the following request body:
    """
    {
      "title": "New Article",
      "content": "This is a test article.",
      "status": "published",
      "imageUrls": ["http://example.com/image.png"]
    }
    """
    When I call API "CreateArticle"
    Then response status should be 200

  @atomic
  Scenario: Get article by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetArticleByID"
    Then response status should be 200

  @atomic
  Scenario: Update an article
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "title": "Updated Article",
      "content": "This is an updated test article.",
      "status": "draft",
      "imageUrls": ["http://example.com/image_updated.png"]
    }
    """
    When I call API "UpdateArticle"
    Then response status should be 200

  @atomic
  Scenario: Delete an article
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteArticle"
    Then response status should be 200

  @atomic
  Scenario: Change article status
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "data": "archived"
    }
    """
    When I call API "ChangeArticleStatus"
    Then response status should be 200
