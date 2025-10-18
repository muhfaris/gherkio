@journey
Feature: API Journey Tests

  Scenario: Full lifecycle of an Announcement
    # 0. Authenticate first
    When I run flow "authenticate-superadmin"
    Given I set auth "bearer"

    # 1. Create a new announcement and capture its ID
    When I run flow "create-announcement-and-capture-id"
    Then the store should not be empty
    And I wait 500ms

    # 2. Use the captured ID to get the specific announcement
    Given I set path params:
      | id | {{.store.createdAnnouncementID}} |
    When I call API "GetAnnouncementByID"
    Then response status should be 200
    And json "name" should equal "Test Journey Announcement"

    # 3. Use the captured ID to update the announcement
    Given I set path params:
      | id | {{.store.createdAnnouncementID}} |
    When I call API "UpdateAnnouncement" with body:
    """
    {
      "name": "Updated Journey Announcement",
      "date": "2024-10-18T10:00:00Z",
      "description": "This announcement has been updated."
    }
    """
    Then response status should be 200

    # 4. Use the captured ID to delete the announcement
    Given I set path params:
      | id | {{.store.createdAnnouncementID}} |
    When I call API "DeleteAnnouncement"
    Then response status should be 200

    # 5. Verify that the announcement is gone
    Given I set path params:
      | id | {{.store.createdAnnouncementID}} |
    When I call API "GetAnnouncementByID"
    Then response status should be 404
