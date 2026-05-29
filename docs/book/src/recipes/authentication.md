# Authentication & JWT Token Recycling Flow

This recipe demonstrates how to authenticate a user dynamically, extract their access token, inspect decoded JWT claims, and inject the token into all subsequent request headers.

---

## 📝 The Scenario

```yaml
scenario: Authenticated User Flow
tags:
  - security
  - authentication

steps:
  # Step 1: Login & obtain access token
  - request:
      method: POST
      url: https://api.my-domain.com/v1/auth/login
      headers:
        Content-Type: application/json
      body:
        username: $accounts.admin.username
        password: $accounts.admin.password
    expect:
      status: 200
      body.token: exists
      body.email: email
      
      # Assert decoded claims from response.token (accessToken)
      jwt.role: super-admin
      jwt.sub: uuid
      jwt.isActive: true
    save:
      # Extract token to $authToken variable
      authToken: body.token

  # Step 2: Use the dynamic token to query secure endpoints
  - request:
      method: GET
      url: https://api.my-domain.com/v1/secure/profile
      headers:
        Authorization: "Bearer ${authToken}"
    expect:
      status: 200
      body.role: super-admin
```

---

## 💡 Key Design Patterns Used

1. **Direct Credentials Extraction**: Sourced securely from environmental configurations via `$accounts.admin.username` to keep secrets out of version control.
2. **Automatic JWT Assertion**: The `jwt.<claim>` keyword triggers standard JWT claim extraction under the hood on response fields matching `token` or `accessToken`.
3. **Dynamic Variable Recycling**: Using the `save:` block to extract the token, then injecting it cleanly as `Authorization: "Bearer ${authToken}"` in next step headers.
