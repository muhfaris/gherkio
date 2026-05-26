package model

import "fmt"

// Credentials represents a credentials file for an environment.
// Contains multiple named accounts that can be used for testing.
type Credentials struct {
	Accounts map[string]Account `yaml:"accounts" jsonschema:"required,description=Map of account names to account credentials"`
}

// Account represents a single set of credentials for an account.
// Username and Password are optional — accounts can be data-only namespaces (e.g. API keys).
type Account struct {
	Username string                 `yaml:"username,omitempty" jsonschema:"description=Account username or email (optional for data-only accounts)"`
	Password string                 `yaml:"password,omitempty" jsonschema:"description=Account password (optional for data-only accounts)"`
	Role     string                 `yaml:"role,omitempty" jsonschema:"description=Account role (e.g. admin, viewer)"`
	Extra    map[string]interface{} `yaml:",inline" jsonschema:"description=Additional custom fields (strings, nested maps, or lists)"`
}

// Validate checks if the credentials file has valid data.
// Accounts without username/password are allowed (e.g. namespaces for API keys).
func (c *Credentials) Validate() error {
	if len(c.Accounts) == 0 {
		return fmt.Errorf("no accounts defined in credentials file")
	}
	return nil
}

// GetAccount returns an account by name.
func (c *Credentials) GetAccount(name string) (Account, bool) {
	account, exists := c.Accounts[name]
	return account, exists
}

// AccountNames returns a list of all account names.
func (c *Credentials) AccountNames() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	return names
}

// ToMap converts all accounts to a nested map suitable for $accounts variable injection.
// Returns map[accountName]map[fieldName]value.
// Only includes non-empty standard fields, allowing accounts that are just data namespaces.
func (c *Credentials) ToMap() map[string]interface{} {
	result := make(map[string]interface{}, len(c.Accounts))
	for name, account := range c.Accounts {
		accountMap := make(map[string]interface{})
		if account.Username != "" {
			accountMap["username"] = account.Username
		}
		if account.Password != "" {
			accountMap["password"] = account.Password
		}
		if account.Role != "" {
			accountMap["role"] = account.Role
		}
		for k, v := range account.Extra {
			accountMap[k] = v
		}
		result[name] = accountMap
	}
	return result
}
