# Credentials File Layout

Manage test account usernames, passwords, API keys, and client secrets securely. Credentials map to target environments and are stored inside `.gherkio/credentials/`.

---

## 🔒 Directory Convention & Security

Credentials files are mapped exactly to environments:

- `--env staging` loads `.gherkio/credentials/staging.yaml` if it exists.

> [!CAUTION]
> Add `.gherkio/credentials/*.yaml` to your project's `.gitignore` file to ensure sensitive client secrets are never committed to version control.

---

## 📝 Credentials YAML Layout

```yaml
# .gherkio/credentials/staging.yaml
accounts:
  # Account Role key
  admin:
    username: admin@company.com
    password: super-secret-key-1
    clientId: cid_12389
    
  # Additional Account Roles
  manager:
    username: manager@company.com
    password: manager-secret-key
    clientId: cid_45678
```

---

## ⚡ Referencing Credentials in DSL

Gherkio makes all variables nested inside an account map immediately available inside test scenarios via two pathways:

### 1. Direct Explicit Variable Binding
Use `$accounts.<role>.<field>` to reference any credential property explicitly:

```yaml
steps:
  - request:
      method: POST
      url: /login
      body:
        email: $accounts.admin.username
        pass: $accounts.admin.password
```

### 2. Contextual Active Role Auto-Injection
If you execute the test run specifying a selected account, Gherkio dynamically flattens the account keys directly into the root context:

```bash
gherkio run my-test.yaml --env staging --account manager
```

Now, inside the test, you can directly reference `$username` and `$password` without specifying the account role prefix:

```yaml
steps:
  - request:
      method: POST
      url: /login
      body:
        email: $username            # Dynamically binds to manager@company.com
        pass: $password            # Dynamically binds to manager-secret-key
```
This is the ultimate pattern for verifying role isolation boundaries with identical test scripts.
