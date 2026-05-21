### Overview
```
.gherkio/
├── config.yaml
└── environments/
    ├── local.yaml
    ├── staging.yaml
    └── production.yaml
├── tests/ -- tests/ HARUS flexible
├── reports/ -- Folder hasil execution.
    ├── latest/
    │   ├── report.html
    │   ├── execution.json
    │   ├── artifacts/
    │   └── logs/
    │
    ├── 2026-05-20_22-30/
    └── archive/
└── schemas/  -- Tempat reusable validation schema.
    ├── auth/
    │   └── login-response.yaml
    │
    ├── items/
    │   ├── item-response.yaml
    │   └── item-list-response.yaml
    │
    └── users/
    └── profile-response.yaml
```
