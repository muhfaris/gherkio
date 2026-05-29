/* ----------------------------------------------------
   Gherkio Interactive Playground: Application Logic
------------------------------------------------------- */

// 📚 1. Documentation Database (Preloaded high-value chapters)
const DOC_DATA = {
    "getting-started": {
        category: "Getting Started",
        chapters: {
            "installation": {
                title: "Installation & Requirements",
                content: `<h2>Requirements</h2>
<p>Gherkio is a zero-dependency self-contained executable. To run it, you only need a standard operating system terminal.</p>
<blockquote><strong>Minimum Specs:</strong> Linux, macOS, or Windows | Go 1.25+ (only if compiling from source)</blockquote>

<h2>Installing via Go Command</h2>
<p>If you have Go installed on your laptop, run the standard compilation fetcher:</p>
<pre><code>go install github.com/muhfaris/gherkio@latest</code></pre>

<h2>Installing via Release Archives</h2>
<p>You can fetch compiled release binaries directly from GitHub Releases. Unpack the tarball and move Gherkio to your execution path:</p>
<pre><code>tar -xzf gherkio_Linux_x86_64.tar.gz
sudo mv gherkio /usr/local/bin/</code></pre>

<h2>Verifying Setup</h2>
<p>Ensure Gherkio is accessible in your environment:</p>
<pre><code>gherkio --version</code></pre>`
            },
            "quickstart": {
                title: "2-Minute Quickstart",
                content: `<h2>Step 1: Scaffolding a Sandbox</h2>
<p>Initialize a testing workspace in an empty directory:</p>
<pre><code>mkdir api-playground && cd api-playground
gherkio init</code></pre>
<p>This generates the standard testing directory tree containing an example login test, a local environment target, and execution configs.</p>

<h2>Step 2: Execute your First Scenario</h2>
<p>Execute the scaffolded login verification test in verbose mode:</p>
<pre><code>gherkio run example/auth/login.yaml --verbose</code></pre>
<p>This runs the YAML test steps, automatically prints pretty terminal tables, masks password parameters, and prints structural assertion outcomes.</p>`
            }
        }
    },
    "dsl-reference": {
        category: "DSL Reference",
        chapters: {
            "matchers": {
                title: "Assertion Matchers Library",
                content: `<p>Gherkio has <strong>25+ built-in assertion matchers</strong> to validate body fields, headers, and JWT claims without writing custom scripts.</p>

<h2>Existence Checks</h2>
<table>
    <tr><th>Keyword</th><th>Syntax Example</th><th>Description</th></tr>
    <tr><td><code>exists</code></td><td><code>body.id: exists</code></td><td>Validates field is not null and present.</td></tr>
    <tr><td><code>not exists</code></td><td><code>body.deletedAt: not exists</code></td><td>Validates field is absent.</td></tr>
</table>

<h2>Data Format Matchers</h2>
<table>
    <tr><th>Keyword</th><th>Syntax Example</th><th>Matches If...</th></tr>
    <tr><td><code>uuid</code></td><td><code>body.id: uuid</code></td><td>Value is a valid UUID v4 format.</td></tr>
    <tr><td><code>email</code></td><td><code>body.email: email</code></td><td>Value conforms to email patterns.</td></tr>
    <tr><td><code>datetime</code></td><td><code>body.createdAt: datetime</code></td><td>Value matches ISO8601/RFC3339 timestamps.</td></tr>
</table>

<h2>Numeric Boundaries</h2>
<table>
    <tr><th>Keyword</th><th>Syntax Example</th><th>Matches If...</th></tr>
    <tr><td><code>gt &lt;num&gt;</code></td><td><code>body.rating: gt 4.2</code></td><td>Value is greater than threshold.</td></tr>
    <tr><td><code>gte &lt;num&gt;</code></td><td><code>body.price: gte 10.0</code></td><td>Value is greater or equal to threshold.</td></tr>
</table>`
            },
            "generators": {
                title: "Dynamic Variable Generators",
                content: `<p>Gherkio includes built-in randomizers that evaluate fresh values per step. This guarantees run thread isolation in multi-threaded executions.</p>

<h2>Available Generators</h2>
<ul>
    <li><code>$uuid</code>: Generates a fresh UUID v4 string.</li>
    <li><code>$ulid</code>: Generates a fresh ULID string.</li>
    <li><code>$randomEmail</code>: Outputs unique user emails (e.g. <code>user_812398@example.com</code>).</li>
    <li><code>$randomPhone</code>: Outputs custom phone strings matching Indonesian standards (e.g. <code>+628...</code>).</li>
    <li><code>$randomInt</code>: Outputs a random positive integer.</li>
    <li><code>\${randomInt(min,max)}</code>: Custom range bounds (e.g. <code>\${randomInt(10,50)}</code>).</li>
</ul>`
            }
        }
    },
    "recipes": {
        category: "Real-World Recipes",
        chapters: {
            "auth-recycle": {
                title: "JWT Token Authentication",
                content: `<p>A complete pattern demonstrating how to retrieve a dynamic token, decode its claims, cache the token, and recycle it in subsequent requests.</p>
<pre><code># Scenario setup
scenario: Authenticate & Query Profile
steps:
  - request:
      method: POST
      url: /v1/login
      body:
        user: $accounts.admin.username
        pass: $accounts.admin.password
    expect:
      status: 200
      jwt.role: super-admin
    save:
      myToken: body.accessToken

  - request:
      method: GET
      url: /v1/secure/profile
      headers:
        Authorization: "Bearer \${myToken}"
    expect:
      status: 200</code></pre>`
            }
        }
    }
};

// 📝 Preset editor examples
const YAML_PRESETS = {
    auth: `scenario: Secure Account Creation Flow
tags:
  - security
  - checkout

steps:
  - request:
      method: POST
      url: /v1/auth/login
      headers:
        Content-Type: application/json
      body:
        username: $accounts.admin.username
        password: $accounts.admin.password
    expect:
      status: 200
      body.token: exists
      jwt.role: super-admin
    save:
      userAuthToken: body.token

  - request:
      method: GET
      url: /v1/users/profile
      headers:
        Authorization: Bearer \${userAuthToken}
    expect:
      status: 200
      body.email: email`,

    polling: `scenario: Asynchronous Order Settlement
tags:
  - async
  - polling

steps:
  - request:
      method: POST
      url: /v1/orders/sync
      body:
        orderId: $uuid
    expect:
      status: 202
    save:
      syncJobId: body.jobId

  - request:
      method: GET
      url: /v1/orders/sync/$syncJobId
    retry:
      attempts: 5
      interval: 1.5s
      backoff: 1.5
    expect:
      status: 200
      body.status: success`,

    bulk: `scenario: Catalog Bulk Creation
tags:
  - bulk
  - catalog

steps:
  - request:
      method: POST
      url: /v1/products/bulk
      body:
        - sku: LAP-CORE-I5
          name: Workstation Laptop
        - sku: LAP-CORE-I7
          name: Developer Workstation
    expect:
      status: 201
      count(body): 2
      all(body.sku): startsWith LAP-`
};

// State Variables
let currentActiveTab = "doc";
let activeDocId = "installation";

// Initialize App
window.addEventListener("DOMContentLoaded", () => {
    // Generate Sidebar Links
    renderSidebar();
    
    // Load first chapter
    loadChapter("getting-started", "installation");
    
    // Load default YAML Visualizer editor example
    document.getElementById("yamlEditor").value = YAML_PRESETS.auth;
    triggerVisualization();

    // Trigger icon rendering
    lucide.createIcons();
});

// Sidebar Navigator Renderer
function renderSidebar() {
    const nav = document.getElementById("docNav");
    nav.innerHTML = "";

    for (const [catKey, catObj] of Object.entries(DOC_DATA)) {
        const catContainer = document.createElement("div");
        catContainer.className = "nav-cat";

        const title = document.createElement("div");
        title.className = "cat-title";
        title.innerText = catObj.category;
        catContainer.appendChild(title);

        for (const [chapKey, chapObj] of Object.entries(catObj.chapters)) {
            const link = document.createElement("a");
            link.className = `nav-link ${chapKey === activeDocId ? "active" : ""}`;
            link.innerText = chapObj.title;
            link.id = `nav-${chapKey}`;
            link.addEventListener("click", () => {
                loadChapter(catKey, chapKey);
                switchMainTab("doc");
            });
            catContainer.appendChild(link);
        }

        nav.appendChild(catContainer);
    }
}

// Chapter Content Loader
function loadChapter(catKey, chapKey) {
    const oldActiveLink = document.querySelector(".nav-link.active");
    if (oldActiveLink) oldActiveLink.classList.remove("active");

    const newActiveLink = document.getElementById(`nav-${chapKey}`);
    if (newActiveLink) newActiveLink.classList.add("active");

    activeDocId = chapKey;
    const chapter = DOC_DATA[catKey].chapters[chapKey];
    
    document.getElementById("docEyebrow").innerText = DOC_DATA[catKey].category;
    document.getElementById("docTitle").innerText = chapter.title;
    document.getElementById("docBody").innerHTML = chapter.content;
}

// Tab Selector Switcher
function switchMainTab(tabId) {
    // Remove active tab headers
    document.querySelectorAll(".tab-btn").forEach(btn => btn.classList.remove("active"));
    // Hide all tab panels
    document.querySelectorAll(".tab-content").forEach(panel => panel.classList.remove("active"));

    // Set active values
    if (tabId === "doc") {
        document.getElementById("tabDoc").classList.add("active");
        document.getElementById("contentDoc").classList.add("active");
    } else if (tabId === "visualizer") {
        document.getElementById("tabVisualizer").classList.add("active");
        document.getElementById("contentVisualizer").classList.add("active");
        triggerVisualization();
    } else if (tabId === "converter") {
        document.getElementById("tabConverter").classList.add("active");
        document.getElementById("contentConverter").classList.add("active");
    }
    
    currentActiveTab = tabId;
}

// Visualizer: Load Preset examples
function loadPresetExample() {
    const selector = document.getElementById("exampleSelector");
    const yaml = YAML_PRESETS[selector.value];
    document.getElementById("yamlEditor").value = yaml;
    triggerVisualization();
}

// Visualizer: Lightweight Gherkio YAML step parser
function triggerVisualization() {
    const text = document.getElementById("yamlEditor").value;
    const flowContainer = document.getElementById("flowOutput");
    const statusBadge = document.getElementById("lintStatus");
    
    flowContainer.innerHTML = "";

    try {
        // Parse step blocks line-by-line in JavaScript
        const parsedSteps = parseGherkioYaml(text);
        
        if (parsedSteps.length === 0) {
            flowContainer.innerHTML = `<div class="helper-text" style="text-align: center; margin-top: 40px;">No steps detected. Type some Gherkio steps inside the editor!</div>`;
            return;
        }

        statusBadge.innerText = "Valid DSL";
        statusBadge.className = "badge success";

        parsedSteps.forEach((step, idx) => {
            // Render Step Card
            const card = document.createElement("div");
            card.className = "step-node";

            // Card Header
            const header = document.createElement("div");
            header.className = "step-header";
            header.onclick = () => toggleStepCollapse(idx);

            const left = document.createElement("div");
            left.className = "step-left";

            const stepIdx = document.createElement("span");
            stepIdx.className = "step-index";
            stepIdx.innerText = idx + 1;
            left.appendChild(stepIdx);

            const methodBadge = document.createElement("span");
            methodBadge.className = `step-method ${step.method.toLowerCase()}`;
            methodBadge.innerText = step.method;
            left.appendChild(methodBadge);

            const urlSpan = document.createElement("span");
            urlSpan.className = "step-url";
            urlSpan.innerText = step.url || step.useFile || "/";
            left.appendChild(urlSpan);

            header.appendChild(left);
            
            const arrowIcon = document.createElement("i");
            arrowIcon.setAttribute("data-lucide", "chevron-down");
            header.appendChild(arrowIcon);
            
            card.appendChild(header);

            // Collapsible Details
            const bodyCollapsed = document.createElement("div");
            bodyCollapsed.className = "step-body-collapsed active";
            bodyCollapsed.id = `step-body-${idx}`;

            // Body Payload section
            if (step.body) {
                const title = document.createElement("div");
                title.className = "block-title";
                title.innerHTML = `<i data-lucide="file-code" style="width: 14px;"></i> Payload JSON`;
                bodyCollapsed.appendChild(title);

                const content = document.createElement("div");
                content.className = "block-content";
                content.innerText = step.body;
                bodyCollapsed.appendChild(content);
            }

            // Assertions section
            if (step.expect.length > 0) {
                const title = document.createElement("div");
                title.className = "block-title";
                title.innerHTML = `<i data-lucide="check-square" style="width: 14px;"></i> Expected Assertions`;
                bodyCollapsed.appendChild(title);

                const content = document.createElement("div");
                content.className = "block-content";
                content.innerText = step.expect.join("\n");
                bodyCollapsed.appendChild(content);
            }

            // Save variables section
            if (step.save.length > 0) {
                const title = document.createElement("div");
                title.className = "block-title";
                title.innerHTML = `<i data-lucide="save" style="width: 14px;"></i> Saves Context Variables`;
                bodyCollapsed.appendChild(title);

                const content = document.createElement("div");
                content.className = "block-content";
                content.innerText = step.save.join("\n");
                bodyCollapsed.appendChild(content);
            }

            // Retry section
            if (step.retry) {
                const title = document.createElement("div");
                title.className = "block-title";
                title.innerHTML = `<i data-lucide="refresh-cw" style="width: 14px;"></i> Polling Retry Config`;
                bodyCollapsed.appendChild(title);

                const content = document.createElement("div");
                content.className = "block-content";
                content.innerText = step.retry;
                bodyCollapsed.appendChild(content);
            }

            card.appendChild(bodyCollapsed);
            flowContainer.appendChild(card);

            // Add animated arrow between steps
            if (idx < parsedSteps.length - 1) {
                const arrow = document.createElement("div");
                arrow.className = "step-arrow";
                arrow.innerHTML = `<i data-lucide="arrow-down" style="width: 20px;"></i>`;
                flowContainer.appendChild(arrow);
            }
        });

        lucide.createIcons();

    } catch (e) {
        statusBadge.innerText = "Syntax Warning";
        statusBadge.className = "badge error";
    }
}

// Helper: Toggle Collapse Panels
function toggleStepCollapse(idx) {
    const body = document.getElementById(`step-body-${idx}`);
    body.classList.toggle("active");
}

// Gherkio Custom Line YAML parser
function parseGherkioYaml(text) {
    const steps = [];
    const lines = text.split("\n");

    let currentStep = null;
    let inRequest = false;
    let inExpect = false;
    let inSave = false;
    let inBody = false;
    let inRetry = false;

    let bodyIndents = [];

    lines.forEach(line => {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith("#")) return;

        // Detect new step in list
        if (line.startsWith("  - request:") || line.startsWith("  - use:")) {
            if (currentStep) steps.push(currentStep);

            currentStep = {
                method: "GET",
                url: "",
                body: "",
                expect: [],
                save: [],
                useFile: "",
                retry: ""
            };

            inRequest = false;
            inExpect = false;
            inSave = false;
            inBody = false;
            inRetry = false;

            if (line.startsWith("  - use:")) {
                currentStep.method = "USE";
                currentStep.useFile = trimmed.replace("- use:", "").trim();
            } else {
                inRequest = true;
            }
            return;
        }

        if (!currentStep) return;

        // Key Transitions
        if (trimmed.startsWith("expect:")) {
            inRequest = false;
            inExpect = true;
            inSave = false;
            inBody = false;
            inRetry = false;
            return;
        }

        if (trimmed.startsWith("save:")) {
            inRequest = false;
            inExpect = false;
            inSave = true;
            inBody = false;
            inRetry = false;
            return;
        }

        if (trimmed.startsWith("retry:")) {
            inRequest = false;
            inExpect = false;
            inSave = false;
            inBody = false;
            inRetry = true;
            return;
        }

        // Inner structures
        if (inRequest) {
            if (trimmed.startsWith("method:")) {
                currentStep.method = trimmed.replace("method:", "").trim();
            } else if (trimmed.startsWith("url:")) {
                currentStep.url = trimmed.replace("url:", "").trim();
            } else if (trimmed.startsWith("body:")) {
                inBody = true;
                const bodyVal = trimmed.replace("body:", "").trim();
                if (bodyVal) {
                    currentStep.body = bodyVal;
                    inBody = false;
                }
            } else if (inBody) {
                // Collect multi-line body sequence
                currentStep.body += (currentStep.body ? "\n" : "") + trimmed;
            }
        } else if (inExpect) {
            currentStep.expect.push(trimmed);
        } else if (inSave) {
            currentStep.save.push(trimmed);
        } else if (inRetry) {
            currentStep.retry += (currentStep.retry ? ", " : "") + trimmed;
        }
    });

    if (currentStep) steps.push(currentStep);
    return steps;
}

// 🔁 4. Interactive Client-Side cURL Converter
function convertCurl() {
    const curl = document.getElementById("curlInput").value.trim();
    const output = document.getElementById("yamlOutput");

    if (!curl) {
        output.value = "# Please type or paste a cURL command first.";
        return;
    }

    // Default parsed parameters
    let method = "GET";
    let url = "/v1/endpoint";
    let headers = [];
    let body = "";

    // 1. Extract Method
    const methodMatch = curl.match(/-X\s+([A-Za-z]+)/) || curl.match(/--request\s+([A-Za-z]+)/);
    if (methodMatch) {
        method = methodMatch[1].toUpperCase();
    } else if (curl.includes("-d ") || curl.includes("--data ") || curl.includes("--data-raw ")) {
        method = "POST";
    }

    // 2. Extract URL
    // Standard matches for https/http URLs
    const urlMatch = curl.match(/https?:\/\/[^\s'"]+/);
    if (urlMatch) {
        url = urlMatch[0];
    }

    // 3. Extract Headers
    const headerRegex = /-H\s+['"]([^'"]+)['"]/g;
    let headerMatch;
    while ((headerMatch = headerRegex.exec(curl)) !== null) {
        headers.push(headerMatch[1]);
    }

    // 4. Extract Body Data
    const bodyMatch = curl.match(/-d\s+['"]([^'"]+)['"]/) || curl.match(/--data(-raw)?\s+['"]([^'"]+)['"]/);
    if (bodyMatch) {
        body = bodyMatch[1] || bodyMatch[2];
        try {
            // Attempt to pretty format JSON if possible
            const parsed = JSON.parse(body);
            body = JSON.stringify(parsed, null, 4);
        } catch (e) {
            // Keep original string if not valid JSON
        }
    }

    // Compile into beautiful Gherkio step block
    let dsl = `  - request:\n`;
    dsl += `      method: ${method}\n`;
    dsl += `      url: ${url}\n`;
    
    if (headers.length > 0) {
        dsl += `      headers:\n`;
        headers.forEach(h => {
            const idx = h.indexOf(":");
            if (idx !== -1) {
                const key = h.slice(0, idx).trim();
                const val = h.slice(idx + 1).trim();
                dsl += `        ${key}: "${val}"\n`;
            }
        });
    }

    if (body) {
        dsl += `      body:\n`;
        const lines = body.split("\n");
        if (lines.length > 1) {
            lines.forEach(l => {
                dsl += `        ${l}\n`;
            });
        } else {
            dsl += `        ${body}\n`;
        }
    }

    dsl += `    expect:\n`;
    dsl += `      status: 200\n`;

    output.value = dsl;
}

// Copy translated YAML script
function copyConvertedYaml() {
    const yaml = document.getElementById("yamlOutput").value;
    navigator.clipboard.writeText(yaml).then(() => {
        alert("✓ Gherkio step copied to clipboard!");
    });
}

// 🌓 Theme controller
const themeBtn = document.getElementById("themeToggle");
themeBtn.addEventListener("click", () => {
    document.body.classList.toggle("light-mode");
    const isLight = document.body.classList.contains("light-mode");
    themeBtn.innerHTML = isLight ? `<i data-lucide="moon"></i>` : `<i data-lucide="sun"></i>`;
    lucide.createIcons();
});
