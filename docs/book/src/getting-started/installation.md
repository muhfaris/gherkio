# Installation

Gherkio is compiled as a single self-contained binary with zero external dependencies. You can install it using Go or by downloading pre-built binaries.

---

## 🛠️ Requirements

- **Operating System**: Linux, macOS, or Windows.
- **Go Version** (if installing from source): Go 1.25+
- **HTTP client**: `curl` (optional, for bidirectional conversion tools).

---

## 📦 Install Options

### 1. Using Go Install (Recommended)

If you have Go installed on your system, run:

```bash
go install github.com/muhfaris/gherkio@latest
```

This compiles the binary and places it under your `$GOPATH/bin` or `$GOBIN` directory. Make sure this directory is in your system's `PATH`.

### 2. Pre-Built Binaries

For systems without Go, download the latest compiled binary from the [GitHub Releases](https://github.com/muhfaris/gherkio/releases) page:

1. Download the archive matching your operating system and architecture (e.g. `gherkio_Linux_x86_64.tar.gz`).
2. Extract the archive:
   ```bash
   tar -xzf gherkio_Linux_x86_64.tar.gz
   ```
3. Move the binary to a directory in your executable path:
   ```bash
   sudo mv gherkio /usr/local/bin/
   ```

### 3. Zero-Install Execution (Go Run)

If you have Go installed but do not want to compile or place the Gherkio binary permanently on your local system path, you can run Gherkio directly from the remote repository using `go run`:

```bash
# Initialize a project sandbox
go run github.com/muhfaris/gherkio@v0.1.0-alpha.3 init

# Execute tests
go run github.com/muhfaris/gherkio@v0.1.0-alpha.3 run .gherkio/tests/example/login.yaml
```

This is the perfect approach for quick trial runs or executing Gherkio inside ephemeral, single-use CI/CD runner pipelines.


---

## ✅ Verifying Installation

Verify that Gherkio is installed correctly by checking its version:

```bash
gherkio --version
```

You should see an output similar to:
```
gherkio version v0.1.0-alpha.3 (commit: 45edd86, built: 2026-05-31, go1.26)
```
