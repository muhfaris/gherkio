# Interactive Browser Playground

To lower Gherkio's learning curve to zero, the repository includes a self-contained, browser-based **Interactive Playground and Documentation Hub** located under `docs/book/playground/index.html`.

You can use the playground to experiment with test scenarios, visualize complex execution paths, and translate legacy commands in real-time.

---

## 🚀 How to Launch the Playground

Since the playground is a static, modern vanilla web application, you can run it instantly:

*   **Online Sandbox**: Access the hosted web workspace directly on **[GitHub Pages Playground](https://muhfaris.github.io/gherkio/playground/index.html)**.
*   **Open via File Manager**: Double-click [docs/book/playground/index.html](../../playground/index.html) to launch it in your default web browser offline.
*   **Open via Terminal**:
    ```bash
    # Linux
    xdg-open docs/book/playground/index.html

    # macOS
    open docs/book/playground/index.html
    ```

---

## 🛠️ Main Features

```
+-------------------------------------------------------------------+
|  [ Read Chapters ]      [ Visual DSL Stepper ]    [ cURL Convert ] |
+-------------------------------------------------------------------+
```

### 1. Visual DSL Stepper
- **What it does**: Type or edit Gherkio declarative YAML test steps on the left pane, and see a live graphical flowchart built on the right pane!
- **Interactive Cards**: Shows step-by-step HTTP methods (GET, POST, etc.), request URL routing, expected assertions, and saved context variables.
- **Timeline Connectors**: Displays step sequence borders with glowing accents.

### 2. cURL-to-YAML Step Translator
- **What it does**: Paste standard legacy cURL terminal logs (e.g., copied from Chrome Developer tools network inspector) and get back perfectly compiled Gherkio steps instantly.
- **Features**: Automatically extracts custom headers, handles multi-line request bodies, parses methods, and includes placeholder 200 expect blocks.
- **Copy Target**: Click "Copy" to drop the translated output straight into your `.gherkio/tests/` files.
