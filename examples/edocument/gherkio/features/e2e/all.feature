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
    And show variable "companyId"
    Given I set path params:
      | id | {{.store.companyId}} |
    When I call API "UpdateCompany" with body:
    """
    {
      "id": "{{.store.companyId}}",
      "name": "Gherkio Corp (Updated) {{fnRandomString 8}}",
      "description": "The primary company configured by our E2E test.",
      "isActive": true
    }
    """
    Then response status should be 204

    # 5. Get the company again and verify the update
    Given I set path params:
      | id | {{.store.companyId}} |
    When I call API "GetCompanyByID"
    Then response status should be 200
    And json "$.data.description" should equal "The primary company configured by our E2E test."

    # 6. Create the Organization Levels hierarchy
    # Level 1: Department
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    {
      "name": "Department {{fnRandomString 8}}"
    }
    """
    Then response status should be 200
    And save "id" as "departmentLevelId"

    # Level 2: Division
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    {
      "name": "Division {{fnRandomString 8}}",
      "parentLevelId": "{{.store.departmentLevelId}}"
    }
    """
    Then response status should be 200
    And save "id" as "divisionLevelId"

    # Level 3: Unit
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    {
      "name": "Unit {{fnRandomString 8}}",
      "parentLevelId": "{{.store.divisionLevelId}}"
    }
    """
    Then response status should be 200
    And save "id" as "unitLevelId"

    # Level 4: Team
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationLevel" with body:
    """
    {
      "name": "Team {{fnRandomString 8}}",
      "parentLevelId": "{{.store.unitLevelId}}"
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "teamLevelId"

    # 7. Create the Organization Nodes
    # -- Departemen Keuangan --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Departemen Keuangan {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.departmentLevelId}}"
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "deptKeuanganId"

    # -- Divisi di bawah Keuangan --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Akuntansi {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptKeuanganId}}"
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "divisiAkuntansiId"

    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Pajak {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptKeuanganId}}"
    }
    """
    Then response status should be 201
    And json "$.data.id" should not be empty
    And save "$.data.id" as "divisiAkuntansiId"

    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Anggaran {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptKeuanganId}}"
    }
    """
    Then response status should be 200
    And json "$.data.id" should not be empty
    And save "$.data.id" as "divisiAnggaranId"

    # -- Departemen SDM --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Departemen SDM {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.departmentLevelId}}"
    }
    """
    Then response status should be 200
    And json "$.data.id" should not be empty
    And save "id" as "deptSdmId"

    # -- Divisi di bawah SDM --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Rekrutmen {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptSdmId}}"
    }
    """
    Then response status should be 200

    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Pengembangan Karyawan {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptSdmId}}"
    }
    """
    Then response status should be 200

    # -- Departemen Teknologi --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Departemen Teknologi {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.departmentLevelId}}"
    }
    """
    Then response status should be 200
    And save "id" as "deptTeknologiId"

    # -- Divisi di bawah Teknologi --
    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Software {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptTeknologiId}}"
    }
    """
    Then response status should be 200

    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi Infrastruktur {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptTeknologiId}}"
    }
    """
    Then response status should be 200

    Given I set path params:
      | companyId | {{.store.companyId}} |
    When I call API "CreateOrganizationNode" with body:
    """
    {
      "name": "Divisi QA {{fnRandomString 8}}",
      "organizationLevelId": "{{.store.divisionLevelId}}",
      "parentOrganizationId": "{{.store.deptTeknologiId}}"
    }
    """
    Then response status should be 200

