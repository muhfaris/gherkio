# Optional Infrastructure Examples

These examples demonstrate features that cannot be emulated by Gherkio's HTTP
mock environment. They are not discovered by the default `.gherkio/tests`
suite, so local development and CI remain self-contained.

## Redis and Redis Sentinel

Copy `.gherkio/environments/redis-example.yaml` to an environment file for your
system, then update its API URL, Redis addresses, Sentinel master name, and
credentials. The scenario uses read-only Redis commands and is safe to adapt to
a shared test environment.

Run the direct Redis example:

```sh
gherkio run .gherkio/examples/redis-cache.yaml --env redis-example
```

To use Sentinel instead, change both `connection` values in the scenario from
`application-cache` to `sentinel-cache`. Scenario syntax is otherwise
identical; selecting direct Redis or Sentinel is an environment concern.
