Feature: Bookings API

  @atomic
  Scenario: Get all bookings
    Given the API is available
    When I call API "GetBookings"
    Then response status should be 200
    And the response body should be a valid JSON

  @atomic
  Scenario: Create a new booking
    Given the API is available
    And I have the following request body:
    """
    {
      "meetingRoomId": "some-room-id",
      "person": 10,
      "purpose": "Project Discussion",
      "requestorName": "Jane Doe",
      "date": "2024-01-01T10:00:00Z",
      "untilDate": "2024-01-01T11:00:00Z",
      "facilities": ["whiteboard"],
      "note": "Urgent meeting."
    }
    """
    When I call API "CreateBooking"
    Then response status should be 200

  @atomic
  Scenario: Get booking by ID
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "GetBookingByID"
    Then response status should be 200

  @atomic
  Scenario: Confirm a booking
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "note": "Your booking is confirmed."
    }
    """
    When I call API "ConfirmBooking"
    Then response status should be 200

  @atomic
  Scenario: Reject a booking
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    And I have the following request body:
    """
    {
      "reason": "The room is under maintenance."
    }
    """
    When I call API "RejectBooking"
    Then response status should be 200

  @atomic
  Scenario: Cancel a booking
    Given the API is available
    And I have the following variables:
      | id | 123e4567-e89b-12d3-a456-426614174000 |
    When I call API "CancelBooking"
    Then response status should be 200
