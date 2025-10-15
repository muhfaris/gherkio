# 🥒 Gherkio

Declarative API testing with Gherkin, reusable flows, and rich HTML reporting.

## Features

- **Gherkin-first**: human-readable scenarios backed by Godog bindings
- **Reusable flows**: compose login/setup sequences without repeating steps
- **`curl` importer**: convert existing curl commands (JSON or multipart/form-data) into catalog + fixtures + feature scaffolds
- **Environment vars**: preload credentials/secrets via `env.vars` with automatic flow fallback
- **Reports**: HTML summary banner with payload drill-down, CSV export, JUnit/Cucumber roadmap

## Installation

You can install gherkio using the install script:

```bash
curl -sSL https://raw.githubusercontent.com/muhfaris/gherkio/main/install.sh | sh
```

Alternatively, you can download the latest release from the [releases page](https://github.com/muhfaris/gherkio/releases).

## Quick Start

```bash
gherkio init
gherkio run --env dev --report html:reports/run.html
```

### Import an API from curl

```bash
gherkio import \
  --api meetingrooms.update \
  --curl "curl https://api.example.com/meeting-rooms/123 \\
    -X PUT -H 'Content-Type: application/json' --data '{"name":"Room"}'"
```

The CLI will replay the request, preview the response, ask whether to generate JSON assertions, and scaffold catalog/fixture/feature files once you confirm.

Need to capture file uploads? Pass your `-F/--form` flags and Gherkio will generate a multipart fixture plus copy the referenced files:

```bash
./gherkio import \
  --api files.upload \
  --curl "curl http://localhost:5069/api/files \\
    -X POST \\
    -H 'accept: */*' \\
    -F 'file=@./fixtures/raw/2025-10-06_08-52.png;type=image/png'"
```


## Documentation

Full documentation (MDX/Nextra) lives in [`docs/`](docs/). Start it locally:

```bash
cd docs
npm install
npm run dev
```

## License

MIT
