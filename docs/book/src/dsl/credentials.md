# Credentials & Multi-Account Testing

Gherkio separates credential configurations from test scripts, enabling multi-role validation across local, staging, and production environments.

---

## 🔑 Directory & File Conventions

Credential maps live under `.gherkio/credentials/<env>.yaml` and match corresponding environments in `.gherkio/environments/<env>.yaml`.

Example: `.gherkio/credentials/staging.yaml`
```yaml
accounts:
  admin:
    username: admin@host.com
    password: admin-secret-key
    role: super-admin
  viewer:
    username: viewer@host.com
    password: viewer-secret-key
    role: reader
```

---

## ⚡ Direct Cross-Account References

You can reference specific account credentials directly inside your YAML request body or headers using the `$accounts.<name>.<field>` notation. This allows cross-account testing without specifying an active user at the command line.

```yaml
steps:
  - request:
      method: POST
      url: /auth/login
      body:
        username: $accounts.admin.username
        password: $accounts.admin.password
    expect:
      status: 200
```

---

## 🏃 Role-Based Command Execution

If you specify an active account flag via the CLI, Gherkio automatically injects all keys for that specific account into the root scope of the test scenario:

```bash
# Injects $username and $password automatically from the 'admin' map
gherkio run tests/my-test.yaml --env staging --account admin
```

You can also run a test suite against all configured accounts sequentially using `--all-accounts`:

```bash
gherkio run tests/my-test.yaml --env staging --all-accounts
```
 Gherkio executes the scenario once for every account defined in the credentials file, ensuring consistent role boundaries.
