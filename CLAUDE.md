# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-sources

Annotates content with the sources it rests on, records who verified each
source or annotation and in what role, and holds source text as embedded
passages for retrieval. Module path
`github.com/aosanya/mwanachama-backend-sources`. Created 2026-10-19.

**A declared domain from the first commit.** The objects come from
`sources.blueprint.json` and the route table from `sources.operations.json`;
storage is `mwanachama-backend-shared/specstore` and the routes are built by
`mwanachama-backend-shared/dispatch.Table`. There are no row structs and no
`AutoMigrate`. `mwanachama-backend-catalog` is the reference;
`mwanachama-backend-insights` was the template for the file layout and
`mwanachama-backend-digitaltwin` for the `dispatch.Table` adapter.

**Standalone library, no server of its own**, and **no auth model**: every
address is gated (`AnonymousActions` is empty) and the mounting host wraps it.

## The rule that matters most: this module names no domain

**A word that means something in one domain and nothing in another does not
belong in this module.** A law firm's sources are statutes and judgments; a
newsroom's are records and interviews. So:

- `source.kind` is free text, not an enum. A domain that filters on it uses
  the `kind` index.
- Custom properties — rights, licence, jurisdiction, confidentiality — live in
  the `properties` JSON document. A domain that filters on one declares an
  index by path (`spec/examples/legal.sources.json` indexes `jurisdiction` and
  `rights`). The module never learns the words.
- The only stored enums are `verification.subject_role` (`source`,
  `annotation` — the module's own roles) and `verification.status`
  (`verified`, `disputed`, `withdrawn`).

`domain_agnostic_test.go` enforces this over identifiers and every declared
field, index and stored enum value. It caught `OperatorRoutes`, copied from
digitaltwin, on day one; the gated half is `GatedRoutes` here.

## Verification identity

`verified_by` comes **from the caller**, never the body. `verifier_name` and
`verifier_role` are **store-owned**: filled by the `ProfileLookup` passed
through `WithProfiles`, overwriting anything the body claimed, and left empty
when no lookup is configured. They are snapshots, so a record keeps reading
correctly after someone changes name or role. A verification with no caller
is refused with `ErrUnverifiedCaller` (401). Verifications are append-only;
the newest is the current standing. See requirements decisions #3–#5.

## Retrieval

`SearchChunks` is brute-force cosine similarity in Go over the chunks that
share the query's `embedding_model`. Every match carries its source and that
source's current verification, so an answer built from it can say where it
came from and who stood behind it. Moving the comparison into Postgres
(pgvector) needs a vector field type in `shared/spec` first — tracked as
SRC4.

## Conventions

- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite via `glebarez/sqlite`) verifies a change.
  `postgres_integration_test.go` (`//go:build integration`, gated on
  `POSTGRES_URL`, `make test-pg`) holds what only Postgres can show: jsonb
  path indexes and the 63-byte identifier limit.
- **Adding a field means editing `sources.blueprint.json` and the Go type in
  `models/` together.** `TestEveryExampleFitsTheTypes` builds a manager over
  every shipped spec.
- **A domain names objects; it does not re-declare them.** A domain spec
  supplies `instance`, the name and table each role lands in, its own indexes,
  and a default on a declared field — nothing else.
- **Assert on `spec.RawNameFor`, never on a physical table name.**
- An address is an entry in `sources.operations.json`; the field docs MCP
  tools need for non-blueprint inputs (filters, `ChunkQuery`) are in
  `routes/tools.go`.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
