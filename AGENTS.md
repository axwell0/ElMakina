Use the Go standard library for collection and data-shape manipulation when it makes code clearer.

Before adding hand-written loops for common slice or map operations, consider packages such as `slices`, `maps`, `cmp`, `iter`, `strings`, and `sort`. Prefer helpers like `slices.Contains`, `slices.Clone`, `slices.SortFunc`, `maps.Clone`, and `maps.Keys` when they reduce repetition without obscuring intent.

Keep explicit loops when they are more readable, when they combine domain logic with the traversal, or when they avoid unnecessary allocations on hot paths.
