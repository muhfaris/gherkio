# Deploying to GitHub Pages

Gherkio's developer documentation is built as a static HTML site using [mdBook](https://github.com/rust-lang/mdBook). Because of this static, zero-runtime architecture, hosting and serving your documentation via **GitHub Pages** is fully supported, completely free, and takes under 2 minutes.

---

## ⚡ Method 1: Automated Deployment via GitHub Actions (Recommended)

The most robust way to host your documentation is by using GitHub Actions. This automates the build process: every time you push edits to the `main` branch, the pipeline regenerates Cobra CLI subcommand manual pages, compiles the static HTML files, and deploys them to GitHub Pages.

### 1. Add the Workflow File
Create a new workflow file at `.github/workflows/deploy-docs.yml` inside your repository:

```yaml
name: Deploy Developer Documentation

on:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v4

      - name: Setup Go Environment
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache: true

      - name: Install mdBook
        run: |
          MDBOOK_VERSION="v0.4.43"
          curl -sSL "https://github.com/rust-lang/mdBook/releases/download/${MDBOOK_VERSION}/mdbook-${MDBOOK_VERSION}-x86_64-unknown-linux-gnu.tar.gz" | tar -xz
          sudo mv mdbook /usr/local/bin/
          mdbook --version

      - name: Build Documentation
        run: |
          make docs-build

      - name: Setup GitHub Pages
        uses: actions/configure-pages@v4

      - name: Upload Static Assets Artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: 'docs/book/book'

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

### 2. Configure GitHub Repository Settings
1. Go to your repository on GitHub.
2. Navigate to **Settings** ➔ **Pages** (under the "Code and automation" section).
3. Under **Build and deployment**, set the **Source** dropdown to **GitHub Actions**.
4. Push your workflow file to `main`. The deployment action will fire, and your live URL will be active within seconds!

---

## 🏗️ Method 2: Manual Local Build & Push

If you prefer to compile the book locally and push the static bundle directly to a dedicated `gh-pages` branch without using GitHub Actions:

### 1. Build the Static HTML
Run the local compilation tool to generate output files under `docs/book/book/`:
```bash
make docs-build
```

### 2. Push to `gh-pages`
You can use standard Git commands or a deployment utility (such as the `gh-pages` npm package or simple scripting) to push the contents of `docs/book/book/` to your `gh-pages` branch:

```bash
# Example quick-script to publish static output
cd docs/book/book
git init
git checkout -b gh-pages
git add .
git commit -m "docs: deploy static developer book manually"
git remote add origin https://github.com/muhfaris/gherkio.git
git push -f origin gh-pages
```

---

## 🎨 Premium Offline Features Compatibility

All custom static page features implemented in Gherkio's mdBook layout — including offline visual **Mermaid Flowcharts** and local assets — work flawlessly when hosted on GitHub Pages:

*   **No External CDNs**: Mermaid rendering relies strictly on the pre-compiled `mermaid.min.js` and `mermaid-init.js` assets bundled inside the workspace. GitHub Pages serves these local files securely under standard HTTPS parameters.
*   **SEO-Friendly Output**: mdBook generates semantic, indexing-friendly static HTML files. Search engines (like Google and DuckDuckGo) can crawl and index your DSL specifications instantly out-of-the-box.
