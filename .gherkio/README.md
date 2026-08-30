# Executable Gherkio Examples

The showcase is organized as a small set of realistic workflows. Each scenario
covers several related DSL features, so the catalog stays readable without
creating one file for every keyword or helper.

Run the self-contained catalog with:

```sh
gherkio validate
gherkio run --env mocked
```

The `mocked` environment requires no external API, database, or Redis server.

| Scenario | Main coverage |
| --- | --- |
| `01_user_profiles.yaml` | examples, composition, conditions, schemas, headers, JWT claims |
| `02_order_cycle.yaml` | setup/teardown, retry, timing, collection projection and type casts |
| `03_upload_avatar.yaml` | multipart files and configurable assets directory |
| `04_collection_matchers.yaml` | repeated query parameters and collection assertions |
| `05_repeat_until.yaml` | bounded multi-step repeat, random objects, object fields, and `count()` |
| `06_dynamic_data_pipeline.yaml` | random/date/string/encoding/hash helpers and saved values |
| `07_contract_matchers.yaml` | scalar, type, network, comparison, negation, and conditional matchers |

Load execution uses the same workflow syntax. For example, two virtual users
each execute the repeat workflow twice:

```sh
gherkio run .gherkio/tests/showcase/05_repeat_until.yaml \
  --env mocked --virtual-users 2 --iterations-per-user 2
```

Examples that require real infrastructure live under `.gherkio/examples/` and
are intentionally excluded from the default suite. See that directory's README
for Redis and Sentinel setup.
