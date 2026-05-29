# Integration Test Recipes

Welcome to the Gherkio Recipe Book! This section compiles **production-grade design patterns** and actionable code templates to solve complex, real-world integration testing challenges.

Rather than simple syntax references, these recipes show how to orchestrate high-performance, resilient, and secure test suites against live microservices.

---

## 🗺️ Recipe Catalogue

Select a recipe below to dive into real-world code templates and design patterns:

| Recipe | Core Use Case | Primary Focus |
| :--- | :--- | :--- |
| **[🔑 Authentication & Token Recycling](authentication.md)** | Dynamic login, JWT claim extraction, and recycling bearer tokens. | Security & Identity |
| **[📦 Bulk Upload & Collections](bulk-upload.md)** | Submitting JSON sequences and asserting conditions across collections (`all()`, `count()`). | Data Pipelines & Lists |
| **[❌ Negative & Error Validation](negative-testing.md)** | Verifying business constraints, bad payloads, and detailed API error shapes. | Contract Resilience |
| **[🔄 Async Polling & Consistency](async-polling.md)** | Handling eventual consistency and long-running tasks using dynamic backoff retries. | Event-Driven Systems |
| **[🛡️ Multi-Account Role Boundaries](multi-account.md)** | Asserting RBAC limitations and security boundaries across different users. | Authorization Security |
| **[📐 Contract Verification with Schemas](schema-validation.md)** | Validating complex JSON bodies against centralized reusable schema rules. | Structural Testing |
| **[⚡ Parallel Execution Safety](parallel-execution.md)** | Orchestrating concurrent test runs securely using dynamic thread-safe parameters. | Performance & Speed |
| **[🏗️ Database Seeding & Data Strategies](data-management.md)** | Setting up and tearing down transient data states to guarantee test isolation. | State Management |

---

## 💡 How to Use These Recipes

1. **Copy & Adapt**: Each recipe includes a fully valid Gherkio DSL YAML snippet. Copy it into your `.gherkio/tests/` directory and replace the mock URLs with your system's endpoints.
2. **Utilize Environments**: Combine these recipes with Gherkio's environment configurations (`.gherkio/environments/`) to run the exact same tests across `local`, `staging`, and `production` targets without code modifications.
3. **Integrate with CI/CD**: Combine **Parallel Execution** and **Data Management** strategies to execute hundreds of validation scenarios inside your GitHub Actions, GitLab CI, or Jenkins pipelines in seconds.
