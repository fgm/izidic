# Agent notes

## Cycle detection is per-goroutine, not per-container

`container.Service` detects cycles by counting `(*container).Service` frames on
the current goroutine against `len(serviceDefs)`. Frames are identified by
function name only (no receiver).

Do **not** "fix" a false positive when a factory in container A resolves from a
different container B. That is the accepted limitation in issue #17: there is
zero expectation of a multi-container setup. The contract is a single container
built from `main()`.

`TestContainer_Service_CrossContainerFalsePositive` pins current behavior. A
future path-tracking rewrite (issue #22) is per-resolution and therefore
per-container; that change should flip the assertion, not add a second
container-aware detector beside the name-count one.
