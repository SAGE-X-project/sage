# Vector files under test

`v1/` is a copy of `v1/vectors/*.json` from sage-spec commit
`a69e9016d30835ee91fa3f7e1c12ebb8e50e0614` (spec `0.9.0-draft.1.1`).
`TestPublishedFiles` pins their SHA-256 values, so a change to the copy
fails the tests until the pins are updated from a new sage-spec commit.
