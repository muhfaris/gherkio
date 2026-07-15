# Quickstart

Start testing your HTTP APIs with Gherkio in under 2 minutes. We will initialize a sandbox project, review the generated declarative scenario, and execute it.

---

## 🚀 1. Initialize a Project

Create a new directory for your test suite and run `gherkio init`:

```bash
mkdir my-api-tests && cd my-api-tests
gherkio init
```

> 💡 **Zero-Install Alternative:** If you do not have the binary installed on your local system path, you can run all Gherkio commands using standard remote Go execution:
> ```bash
> go run github.com/muhfaris/gherkio@v0.1.0-alpha.6 init
> ```

This scaffolds the canonical Gherkio folder structure:
```
.gherkio/
├── config.yaml
├── credentials/
│   └── local.yaml
├── environments/
│   └── local.yaml
├── schemas/
│   └── example/
│       ├── login-response.yaml
│       └── user-response.yaml
└── tests/
    └── example/
        ├── accounts/
        │   └── login-as-alice.yaml
        ├── auth/
        │   ├── login.yaml
        │   ├── me.yaml
        │   └── refresh.yaml
        └── builtins/
            └── login-with-generators.yaml
```

---

## 📝 2. Review the Example Test

Gherkio created a pre-configured example test under `.gherkio/tests/example/auth/login.yaml`. Open it to see the declarative YAML DSL:

```yaml
scenario: login example

steps:
  - request:
      method: POST
      url: /auth/login

      body:
        username: $username
        password: $password
        expiresInMins: 30

    expect:
      status: 200
      body.accessToken: exists
      body.refreshToken: exists
      body.email: email
      schema: example/login-response

    save:
      accessToken: body.accessToken
      refreshToken: body.refreshToken
```

---

## 🏃 3. Run the Test Scenario

Run the scaffolded test using `gherkio run`:

```bash
gherkio run example/auth/login.yaml --verbose
```

> 💡 **Zero-Install Alternative:** You can run Gherkio dynamically without installing using `go run`:
> ```bash
> go run github.com/muhfaris/gherkio@v0.1.0-alpha.6 run example/auth/login.yaml --verbose
> ```

The `--verbose` flag shows full request and response payloads with automatically masked credentials.

You will see the step-by-step console printer output and a final test summary:
```
✔ Step 1: POST https://dummyjson.com/auth/login [200 OK] (182ms)
  ✔ Assertion: status == 200
  ✔ Assertion: body.accessToken exists
  ✔ Assertion: body.refreshToken exists
  ✔ Assertion: body.email matches format email
  ✔ Assertion: response conforms to schema: example/login-response
  → Saved accessToken: eyJhbGciOi...
  → Saved refreshToken: rGhyTk83...

=======================================================
SCENARIO RESULT: PASSED
Total Steps: 1 | Passed: 1 | Failed: 0 | Duration: 182ms
=======================================================
```
