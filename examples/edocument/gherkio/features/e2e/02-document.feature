@e2e
Feature: Document Type Management

  Scenario: Superadmin creates and verifies a new document type
    # 1. Login as superadmin
    When I run flow "authenticate-superadmin"
    Then the store should contain "access_token"
    Given I set auth "bearer"

    # 2. Create a new document type
    When I call API "CreateDocumentType" with body:
    """
    {
      "code": "DOC-{{fnRandomString 8}}",
      "name": "Surat Perjanjian Kerjasama {{fnRandomString 8}}",
      "description": "Dokumen untuk perjanjian kerjasama dengan pihak ketiga.",
      "reminder": 30,
      "emailReceivers": [
        "akunsosmedx01@gmail.com"
      ]
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "documentTypeId"

    # 3. Get the document type by ID, save its dynamic values, and verify all details
    Given I set path params:
      | documentTypeId | {{.store.documentTypeId}} |
    When I call API "GetDocumentTypeByID"
    Then response status should be 200
    And save "$.data.code" as "documentTypeCode"
    And save "$.data.name" as "documentTypeName"
    And json "data.code" should equal store "documentTypeCode"
    And json "data.name" should equal store "documentTypeName"
    And json "data.description" should equal "Dokumen untuk perjanjian kerjasama dengan pihak ketiga."
    And json "data.reminders" should exist
    And json 'data.emailReceivers.0' should equal "akunsosmedx01@gmail.com"

    # 4. Verify the new document type also exists in the main list
    When I call API "GetDocumentTypes"
    Then response status should be 200
    And json 'data.#(id=="{{.store.documentTypeId}}")' should exist
    And json 'data.#(id=="{{.store.documentTypeId}}").code' should equal store "documentTypeCode"
    And json 'data.#(id=="{{.store.documentTypeId}}").name' should equal store "documentTypeName"

  Scenario: Superadmin manages document locations
    # 1. Login is already done from the previous scenario, but we re-authenticate for idempotency
    When I run flow "authenticate-superadmin"
    Then the store should contain "access_token"
    Given I set auth "bearer"

    # 2. Create a new document location
    When I call API "CreateDocumentLocation" with body:
    """
    {
      "code": "LOC-{{fnRandomString 8}}",
      "name": "Gudang Arsip Pusat {{fnRandomString 8}}",
      "description": "Lokasi pusat untuk semua arsip fisik."
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "documentLocationId"

    # 3. Get the new location by ID and verify its details
    Given I set path params:
      | documentLocationId | {{.store.documentLocationId}} |
    When I call API "GetDocumentLocationByID"
    Then response status should be 200
    And save "$.data.code" as "documentLocationCode"
    And save "$.data.name" as "documentLocationName"
    And json "data.description" should equal "Lokasi pusat untuk semua arsip fisik."

    # 4. Verify the new location exists in the main list
    When I call API "GetDocumentLocations"
    Then response status should be 200
    And json 'data.#(id=="{{.store.documentLocationId}}").code' should equal store "documentLocationCode"
    And json 'data.#(id=="{{.store.documentLocationId}}").name' should equal store "documentLocationName"

    # 5. Update the document location
    Given I set path params:
      | documentLocationId | {{.store.documentLocationId}} |
    When I call API "UpdateDocumentLocation" with body:
    """
    {
      "code": "{{.store.documentLocationCode}}",
      "name": "Gudang Arsip Cabang {{fnRandomString 8}}",
      "description": "Lokasi cabang untuk arsip regional."
    }
    """
    Then response status should be 204

    # 6. Get the location by ID again and verify the update
    Given I set path params:
      | documentLocationId | {{.store.documentLocationId}} |
    When I call API "GetDocumentLocationByID"
    Then response status should be 200
    And save "$.data.name" as "updatedDocumentLocationName"
    And json "data.description" should equal "Lokasi cabang untuk arsip regional."
    And json "data.code" should equal store "documentLocationCode"

    # 7. Verify the updated location in the main list
    When I call API "GetDocumentLocations"
    Then response status should be 200
    And json 'data.#(id=="{{.store.documentLocationId}}").name' should equal store "updatedDocumentLocationName"
