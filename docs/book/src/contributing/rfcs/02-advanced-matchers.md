# RFC-02: Advanced Matchers

**Status:** Implemented

Extended Gherkio's assertion engine beyond simple equality checks to support type validators, format validators, string pattern matchers, and collection matchers.

**Key decisions:**
- Matchers use a keyword-based syntax: `body.field: uuid`, `body.field: contains foo`
- Collection matchers use function-call syntax: `count(body.items): 3`, `all(body.items.status): active`
- Type matchers (`string`, `number`, `boolean`, `array`, `object`) validate JSON types
- Format matchers (`uuid`, `email`, `datetime`, `uri`, `ipv4`, `ipv6`) validate string formats
