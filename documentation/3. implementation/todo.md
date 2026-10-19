# Task board — pending

| ID | Task | Status | Notes |
|---|---|---|---|
| SRC2 | Mount in `mwanachama-wakala-api`: `go.mod` require + replace, `NewSourceManager` beside the other managers with a `ProfileLookup` backed by actor (name + role of the caller's assignment), routes under the gated mux, `mcp.RegisterTools` on `/mcp` | ⏳ Proposed | Sibling repo; needs the owner's go-ahead. The profile lookup is the piece that makes "verified by Wanjiru Kamau, Senior Partner" real. |
| SRC3 | Gateway active-migration SQL mirror for the four tables | ⏳ Proposed | So `cmd/migrate up` alone provisions a fresh DB. Generate from `cmd/ddl`. |
| SRC4 | Vector search in Postgres: a `vector` field type in `shared/spec` (pgvector `vector(n)`, sqlite fallback to JSON) and `SearchChunks` ordering by `<=>` | ⏳ Proposed | Shared change. Today's Go cosine scan reads every chunk of the model on each search. |
| SRC5 | Rights enforcement: let `permissions` read `properties.rights` / `confidentiality` to refuse export or limit readers | 💭 Open | Needs a decision on which properties are enforceable. Descriptive only for now (decision #8). |
| SRC6 | Embedding producer: a host-side job that splits a filed source's `content_ref` into chunks and embeds them | 💭 Open | Which model, and where it runs, is the owner's call. |
| SRC7 | Create the GitHub repo `aosanya/mwanachama-backend-sources` and push | ⏳ Proposed | Local repo initialised, nothing committed or pushed. |
| SRC8 | An `mcp/` test that calls the tools through a real MCP client, as insights does | ⏳ Proposed | `TestEveryOperationIsATool` covers the build today. |
