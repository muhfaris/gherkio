# 🥒 Gherkio

Declarative API testing with Gherkin, reusable flows, and rich HTML reporting.

## Features

- **Gherkin-first**: human-readable scenarios backed by Godog bindings
- **Reusable flows**: compose login/setup sequences without repeating steps
- **`curl` importer**: convert existing curl commands into catalog + fixtures + feature scaffolds
- **Environment vars**: preload credentials/secrets via `env.vars` with automatic flow fallback
- **Reports**: HTML summary banner with payload drill-down, CSV export, JUnit/Cucumber roadmap

## Quick Start

```bash
go mod tidy
./gherkio init
./gherkio run --env dev --report html:reports/run.html
```

### Import an API from curl

```bash
./gherkio import curl \
  --api meetingrooms.update \
  --curl "curl https://api.example.com/meeting-rooms/123 \\
    -X PUT -H 'Content-Type: application/json' --data '{"name":"Room"}'"
```

The CLI will replay the request, preview the response, ask whether to generate JSON assertions, and scaffold catalog/fixture/feature files once you confirm.

## Documentation

Full documentation (MDX/Nextra) lives in [`docs/`](docs/). Start it locally:

```bash
cd docs
npm install
npm run dev
```

## License

MIT
