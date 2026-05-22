package model

import "fmt"

// Credentials represents a credentials file for an environment.
// Contains multiple named accounts that can be used for testing.
type Credentials struct {
	Accounts map[string]Account `yaml:"accounts"`
}

// Account represents a single set of credentials for an account.
type Account struct {
	Username string            `yaml:"username"`
	Password string            `yaml:"password"`
	Role     string            `yaml:"role,omitempty"`
	Extra    map[string]string `yaml:",inline"`
}

// Validate checks if the credentials file has valid data.
func (c *Credentials) Validate() error {
	if len(c.Accounts) == 0 {
		return fmt.Errorf("no accounts defined in credentials file")
	}
	for name, account := range c.Accounts {
		if account.Username == "" || account.Password == "" {
			return fmt.Errorf("account '%s' is missing username or password", name)
		}
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