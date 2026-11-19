# Hub import drift

What stood between `mwanachama-backend-sources` and the hub's in-tree copy (`projects/hub-steward/code/wakala-api/modules/sources`) before the hub imports this module and deletes the copy.

## Method

This repository has no `scripts/hub-copy-diff.sh` and no `hub-copy-diff.md`, so the comparison was made by hand with the same idea the other sibling modules use: every file of the hub copy, with `wakalaserver/modules/sources` rewritten to `github.com/aosanya/mwanachama-backend-sources` and `wakalaserver/modules/shared` rewritten to `github.com/aosanya/mwanachama-backend-shared`, compared byte for byte (`cmp`) with the file of the same path here; then `diff -rq` of the two trees for files on one side only.

Run on 2026-11-19 against the working tree of the hub's `dev` branch at `43f22b26`. The module folder has no uncommitted change there; its last commit is `f6e1606d` (tests pin that the caller, not the request body, is recorded as a source's filer, an annotation's maker and a verification's voucher). This repository was clean on `main` at `92576ff`.

## Result of the run

Every one of the 23 non-test files of the hub copy is identical to this repository's file of the same path once the two import prefixes are rewritten. This includes `source.go`, `annotation.go`, `chunk.go`, `document.go`, `verification.go`, `search.go`, `manager.go`, `identity_owner.go`, `blueprint.go`, `validate.go`, `store.go`, `provision.go`, `errors.go`, `operations.go`, `patterns.go`, the three JSON files, `models/`, `mcp/` and `routes/`. The hub copy carries no behaviour this repository lacks, and this repository carries no behaviour the hub copy lacks. The only difference in content is one hub-only test file.

## Only in the hub

| Item | What it is | Disposition |
|---|---|---|
| `routes/author_tool_test.go` | Two tests of who signs a record. The tool schemas of `sources_source_create`, `sources_annotation_create` and `sources_verification_create` do not offer `created_by` or `verified_by`; a forged `created_by` or `verified_by` argument is refused before the manager is called; an honest call is filed and verified under the caller | Moved with the code, import paths rewritten, in package `routes` beside `routes_test.go` (package `routes_test`). Passes on the unchanged module code. |

## Only in this repository

| Item | Disposition |
|---|---|
| Test files the hub copy never had: `annotation_test.go`, `chunk_test.go`, `document_test.go`, `domain_agnostic_test.go`, `examples_test.go`, `identity_owner_test.go`, `operations_refers_test.go`, `postgres_integration_test.go`, `source_test.go`, `testdb_test.go`, `verification_test.go`, `vocabulary_test.go`, `mcp/resolve_test.go`, `mcp/resolving_cover_test.go`, `mcp/tool_conversation_meta_test.go`, `routes/routes_test.go` | Keep. The hub copy has no tests apart from the one above. |
| `spec/examples/legal.sources.json`, `spec/examples/newsroom.sources.json` | Keep. Example domain specs that `examples_test.go` builds a manager over. |
| `cmd/ddl`, `documentation/`, `README.md`, `CLAUDE.md`, `Makefile`, `LICENSE`, `go.mod`, `go.sum`, `.gitignore` | Keep. Repository files, outside the module. |

## Differing beyond the import path

None. No operation behaves differently on the two sides, so there is nothing to put to the owner as a question.

## Hub-only wiring outside the module

These stay in the hub and are not moved. When the hub imports this module they keep working by changing only their import path.

| Where in the hub | What it does |
|---|---|
| `internal/api/http/sources_routes.go` | Mounts the module's route table under `/instances/{instanceID}/sources`, resolves the instance's source manager, takes the caller from the session and applies the role's permissions |
| `internal/api/http/mcp_sources_permissions.go` | Registers the resolved MCP tools and guards each tools/call against the caller's role permissions |
| `internal/api/http/mcp.go`, `internal/api/http/mcp_tool_actions.go`, `internal/api/http/membership.go` | Resolve the source manager per instance and list the tool-to-action map |
| `internal/api/http/approvals_apply.go` | Applies an approved `sources_document_create` through `AddDocument` |
| `internal/agencies/registry.go`, `internal/agencies/caches.go`, `internal/agencies/mounts.go`, `internal/accounts/router.go` | Build and pool one `SourceManager` per instance through `SpecForMount`, `Provision` and `NewSourceManager`, under the mount name `sources` |
| `modules/agency/models/permissions_catalog.json` | The `document.search` and `document.read` permissions and the `sources.*` actions they grant |

## Pinned counts

Nothing was merged beyond the one test file, so no pinned count changed.
