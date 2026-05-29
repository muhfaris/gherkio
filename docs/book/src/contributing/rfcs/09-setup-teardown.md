# RFC-09: Setup & Teardown Steps

**Status:** Implemented

Added `setup` and `teardown` blocks to scenarios for pre-condition and post-condition steps.

**Key decisions:**
- Setup failure skips main steps but teardown still runs
- Teardown always runs regardless of setup/steps outcome
- Teardown failures are recorded but don't affect pass/fail
- Steps are tagged with their role (`setup`, `steps`, `teardown`) in output
