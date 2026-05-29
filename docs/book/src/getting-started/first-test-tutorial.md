# Tutorial: Build Your First Test

In this tutorial, you will write a complete Gherkio declarative integration test scenario from scratch. 

We will build a multi-step user journey mimicking a content platform workflow:
1. **Fetch a post** from an API endpoint.
2. **Assert** the response status, check that the title is a non-empty string, and verify the post ID is correct.
3. **Save** the `userId` from the post response.
4. **Fetch the user's profile** using the saved variable and assert their username matches.

---

## 🛠️ Step 1: Create Your Test File

First, navigate to your Gherkio test directory (created during `gherkio init`) and create a new YAML file:

```bash
touch .gherkio/tests/example/auth/my-first-test.yaml
```

Open this file in your favorite text editor.

---

## 📝 Step 2: Define Scenario Metadata

Every Gherkio test suite begins with a scenario declaration and optional tag lists for easy test suite filtering:

```yaml
scenario: Fetch Post and Verify Author Profile
tags:
  - regression
  - tutorial
```

---

## 🔄 Step 3: Write the First Step (Fetch & Assert)

Let's append our first test step under the `steps` list. This step issues a standard HTTP `GET` request to fetch Post `#1`:

```yaml
steps:
  - request:
      method: GET
      url: https://jsonplaceholder.typicode.com/posts/1
    expect:
      status: 200
      body.id: 1
      body.title: string
      body.userId: integer
```

### 💡 What's Happening Here?
- **`request.method` & `request.url`**: Define the HTTP command and fully-qualified target path.
- **`expect.status: 200`**: Ensures the server replies with a `200 OK` status.
- **`expect.body.id: 1`**: Asserts exact equality. The post ID must be exactly `1`.
- **`expect.body.title: string`**: Evaluates the data type. Asserts that the title field returned is a string.
- **`expect.body.userId: integer`**: Ensures the author reference is a valid number.

---

## 💾 Step 4: Extract and Save Context Variables

Before fetching the user's profile, we need to know who wrote Post #1. We can extract the `userId` field dynamically and save it under a temporary variable name:

```yaml
    save:
      authorId: body.userId
```

### 💡 What's Happening Here?
- **`save`**: The context extractor block.
- **`authorId: body.userId`**: Resolves the JSON body field `userId` (e.g. `1`), and binds its value to the session variable `authorId` for subsequent steps.

---

## 🔗 Step 5: Write the Second Step (Variable Reuse)

Now, let's add a second step to retrieve the profile of the author we just saved. We will interpolate the `$authorId` variable directly in the request URL:

```yaml
  - request:
      method: GET
      url: https://jsonplaceholder.typicode.com/users/$authorId
    expect:
      status: 200
      body.id: integer
      body.username: string
```

### 💡 What's Happening Here?
- **`url: .../users/$authorId`**: Gherkio automatically replaces `$authorId` with the value extracted in the previous step (e.g. `https://jsonplaceholder.typicode.com/users/1`).
- **`expect.body.username: string`**: Confirms that the profile contains a valid, non-empty username.

---

## 🚀 Step 6: Review the Full Scenario File

Here is what your completed `.gherkio/tests/example/auth/my-first-test.yaml` file should look like:

```yaml
scenario: Fetch Post and Verify Author Profile
tags:
  - regression
  - tutorial

steps:
  - request:
      method: GET
      url: https://jsonplaceholder.typicode.com/posts/1
    expect:
      status: 200
      body.id: 1
      body.title: string
      body.userId: integer
    save:
      authorId: body.userId

  - request:
      method: GET
      url: https://jsonplaceholder.typicode.com/users/$authorId
    expect:
      status: 200
      body.id: integer
      body.username: string
```

---

## 🏃 Step 7: Run Your Test

Now run your new declarative scenario using the Gherkio CLI:

```bash
gherkio run example/auth/my-first-test.yaml --verbose
```

### 🎉 The Output
Gherkio will execute both steps sequentially, resolving variables on the fly, and outputting beautiful execution metrics:

```
✔ Step 1: GET https://jsonplaceholder.typicode.com/posts/1 [200 OK] (95ms)
  ✔ Assertion: status == 200
  ✔ Assertion: body.id == 1
  ✔ Assertion: body.title matches type string
  ✔ Assertion: body.userId matches type integer
  → Saved authorId: 1

✔ Step 2: GET https://jsonplaceholder.typicode.com/users/1 [200 OK] (87ms)
  ✔ Assertion: status == 200
  ✔ Assertion: body.id matches type integer
  ✔ Assertion: body.username matches type string

=======================================================
SCENARIO RESULT: PASSED
Total Steps: 2 | Passed: 2 | Failed: 0 | Duration: 182ms
=======================================================
```

You've successfully built, written, and verified your first multi-step integration test flow! Next, check out how to structure your custom local environment files to avoid hardcoding domain URLs under **[Folder & Project Setup](project-setup.md)**.
