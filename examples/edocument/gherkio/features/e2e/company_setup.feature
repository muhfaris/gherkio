@e2e
Feature: Company Setup and Configuration Journey

  Scenario: Superadmin configures the default company
    # 1. Login as superadmin
    When I run flow "authenticate-superadmin"
    Then the store should contain "access_token"
    Given I set auth "bearer"

    # 2. Check personal info and privileges
    When I call API "MyInfo"
    Then response status should be 200
    And json "data.roles" should not be empty

    # 3. Get settings and capture the default company ID
    When I call API "GetSettings"
    Then response status should be 200
    And json "data.company.id" should not be empty
    And json "data.company.name" should not be empty
    And save "data.company.id" as "companyId"

    # 4. Update the company details
    Given I set path params:
      | id | {{.store.companyId}} |
    When I call API "UpdateCompany" with body:
    """
    {
      "id": "{{.store.companyId}}",
      "name": "Gherkio Corp (Updated)",
      "description": "The primary company configured by our E2E test.",
      "isActive": true
    }
    """
    Then response status should be 200

    # 5. Get the company again and verify the update
    Given I set path params:
      | id | {{.store.companyId}} |
    When I call API "GetCompanyByID"
    Then response status should be 200
    And json "name" should equal "Gherkio Corp (Updated)"
    And json "description" should equal "The primary company configured by our E2E test."

    # 6. Create the Organization Levels hierarchy
    # Level 1: Department
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    { "name": "Department" }
    """
    Then response status should be 200
    And save "id" as "departmentLevelId"

    # Level 2: Division
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    { "name": "Division", "parentLevelId": "{{.store.departmentLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "divisionLevelId"

    # Level 3: Unit
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    { "name": "Unit", "parentLevelId": "{{.store.divisionLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "unitLevelId"

    # Level 4: Team
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    { "name": "Team", "parentLevelId": "{{.store.unitLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "teamLevelId"

    # 7. Create the Organization Nodes
    # -- Departemen Keuangan --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Departemen Keuangan", "organizationLevelId": "{{.store.departmentLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "deptKeuanganId"

    # -- Divisi di bawah Keuangan --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Akuntansi", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptKeuanganId}}" }
    """
    Then response status should be 200

    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Pajak", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptKeuanganId}}" }
    """
    Then response status should be 200

    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Anggaran", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptKeuanganId}}" }
    """
    Then response status should be 200

    # -- Departemen SDM --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Departemen SDM", "organizationLevelId": "{{.store.departmentLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "deptSdmId"

    # -- Divisi di bawah SDM --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Rekrutmen", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptSdmId}}" }
    """
    Then response status should be 200

    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Pengembangan Karyawan", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptSdmId}}" }
    """
    Then response status should be 200

    # -- Departemen Teknologi --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Departemen Teknologi", "organizationLevelId": "{{.store.departmentLevelId}}" }
    """
    Then response status should be 200
    And save "id" as "deptTeknologiId"

    # -- Divisi di bawah Teknologi --
    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Software", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptTeknologiId}}" }
    """
    Then response status should be 200

    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi Infrastruktur", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptTeknologiId}}" }
    """
    Then response status should be 200

    Given I set query params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    { "name": "Divisi QA", "organizationLevelId": "{{.store.divisionLevelId}}", "parentOrganizationId": "{{.store.deptTeknologiId}}" }
    """
    Then response status should be 200
