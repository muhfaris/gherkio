Feature: Users API

  @atomic
  Scenario: Get all users
    Given the API is available
    When I call API "GetUsers"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new user
    Given the API is available
    And I have the following request body:
    """
    {
      "username": "newuser",
      "email": "newuser@example.com",
      "password": "password",
      "displayName": "New User",
      "isActive": true,
      "role": "some-role-id"
    }
    """
    When I call API "CreateUser"
    Then response status should be 200

  @atomic
  Scenario: Get user by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetUserByID"
    Then response status should be 200

  @atomic
  Scenario: Update a user
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "username": "updateduser",
      "email": "updateduser@example.com",
      "displayName": "Updated User",
      "isActive": false,
      "role": "some-other-role-id"
    }
    """
    When I call API "UpdateUser"
    Then response status should be 200

  @atomic
  Scenario: Delete a user
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "DeleteUser"
    Then response status should be 200
