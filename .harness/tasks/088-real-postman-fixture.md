# 088 Real Postman fixture

`postman/testdata/collection.json` has an invented `_postman_id` (`...-0000-4000-8000-000000000001`). Fixtures are real captured output.

Scope
1. Ask the user for a real export (trimmed, no secrets) or capture one.
2. Replace the fixture; adjust expectations only where real output differs.

Acceptance: fixture is real; `make check` green.
