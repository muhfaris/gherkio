# RFC-20: Parameterized Generators

**Status:** Implemented

Extended the built-in variable generators to support runtime parameters using function-call syntax.

**Key decisions:**
- `${randomInt(min,max)}` syntax for custom range random integers
- Function-call syntax `${func(arg1,arg2)}` is extensible for future generators
- Built-in generators regenerate per step for unique values per request
