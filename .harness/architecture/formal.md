## formal/: claims about the real code

`formal/<area>/` holds test files only, one package per area (`escape`, later `sending`,
`connection`). Each states claims in domain words (`claims_test.go`), drives the real feature
packages and says how strong the evidence is, using `gascoigne-formal`. The zone `formal` may import
base, feature roots and controllers, from test files only (`e2e/layers_rules_test.go`).

| Evidence | Meaning here |
|---|---|
| verified | every state inside the printed space, where the spec's state is the real value and each action calls the real method (`escape`: all 2^layers sets of `overlay.Open`) |
| bounded | a search limit stopped it; the result says so |
| tested | the real code run on generated or walked cases (`Conform`, `ForAll`, `Holds`), shrunk and replayable by seed; never a proof |

- A model is only an oracle small enough to read at a glance; it never re-implements the code. Where
  a model would copy the code, a claim over generated inputs is used instead.
- Every spec has broken variants (a stand-in with the bug it guards against); the test asserts the
  check finds each at the named property. That is the test-first step for code that already exists.
- `make test-formal` runs `formal check ./formal/...`: the claim table with its evidence, and the
  `formal.lock` of each package. Change a claim sentence, then `formal accept -m "<reason>"`; first
  write a lock with `go test ./formal/<area> -run TestClaimsMatchTheirLock -update`.
- The library is private: set `GOPRIVATE=github.com/gascoigne-ai/*` and give git ssh access.
