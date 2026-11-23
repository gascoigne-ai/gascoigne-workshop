## formal/: claims about the real code

`formal/<area>/` holds test files only, one package per area: `escape` (overlay Escape and keyboard
rules, verified over all layer sets), `sending` (`sends.Sends`: one send per owner, late answers
dropped, numbers, cancel once; `Conform` against a small oracle) and `connection` (`tabs.Connection`:
refused events, session, user close, retry, stale connects, leaving Connecting; verified on the
real value). Each states claims in domain words (`claims_test.go`), drives the real feature
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
- `Conform` walks end with chance one half at each step; set `MinSteps` so a bug that needs a run of
  steps (begin, cancel, begin, late answer) is reached. Each broken use of the real code is a test
  that the walk diverges, which also shows the walk is long enough.
