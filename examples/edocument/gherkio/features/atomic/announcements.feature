Feature: Announcements API

  @atomic
  Scenario: Get all announcements
    Given the API is available
    When I call API "GetAnnouncements"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new announcement
    Given the API is available
    And I have the following request body:
    """
    {
      "name": "New Announcement",
      "date": "2024-01-01T00:00:00Z",
      "untilDate": "2024-01-31T23:59:59Z",
      "imageUrls": ["http://example.com/image.png"],
      "description": "This is a test announcement."
    }
    """
    When I call API "CreateAnnouncement"
    Then response status should be 200

  @atomic
  Scenario: Get announcement by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetAnnouncementByID"
    Then response status should be 200

  @atomic
  Scenario: Update an announcement
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "name": "Updated Announcement",
      "date": "2024-01-01T00:00:00Z",
      "untilDate": "2024-01-31T23:59:59Z",
      "imageUrls": ["http://example.com/image_updated.png"],
      "description": "This is an updated test announcement."
    }
    """
    When I call API "UpdateAnnouncement"
    Then response status should be 200

  @atomic
  Scenario: Delete an announcement
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteAnnouncement"
    Then response status should be 200
