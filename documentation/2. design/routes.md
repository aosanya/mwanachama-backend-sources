# The route table is declared

Every address is an entry in
[`sources.operations.json`](../../sources.operations.json).
`routes/routes.go` hands it, the sentinel map and the empty anonymous
allowlist to `dispatch.NewTable`. `routes/tools.go` builds MCP tools from the
same table, and `mcp/mcp.go` registers them. One table, two surfaces.

| Method | Path | Call | Action | Status |
|---|---|---|---|---|
| POST | `/sources` | `CreateSource` | `sources.source.create` | 201 |
| GET | `/sources` | `ListSources` | `sources.source.list` | 200 |
| GET | `/sources/{sourceID}` | `GetSource` | `sources.source.read` | 200 |
| PUT | `/sources/{sourceID}` | `UpdateSource` | `sources.source.update` | 200 |
| POST | `/annotations` | `CreateAnnotation` | `sources.annotation.create` | 201 |
| GET | `/annotations` | `ListAnnotations` | `sources.annotation.list` | 200 |
| GET | `/annotations/{annotationID}` | `GetAnnotation` | `sources.annotation.read` | 200 |
| DELETE | `/annotations/{annotationID}` | `DeleteAnnotation` | `sources.annotation.delete` | 204 |
| POST | `/verifications` | `Verify` | `sources.verification.create` | 201 |
| GET | `/verifications` | `ListVerifications` | `sources.verification.list` | 200 |
| POST | `/sources/{sourceID}/chunks` | `AddChunk` | `sources.chunk.create` | 201 |
| GET | `/sources/{sourceID}/chunks` | `ListChunks` | `sources.chunk.list` | 200 |
| POST | `/chunks/search` | `SearchChunks` | `sources.chunk.search` | 200 |

## Who outranks the body

- **The caller:** `created_by` on sources and annotations, and `verified_by`
  on verifications. The same binding keeps these fields out of every MCP tool
  schema.
- **The address:** the source id on `PUT /sources/{sourceID}` and on
  `POST /sources/{sourceID}/chunks`.
- **The store:** `verifier_name` and `verifier_role` are overwritten by the
  profile lookup.

`TestTheCallerOutranksTheBodyOnVerify` and
`TestTheAddressOutranksTheBodyOnAChunk` hold these.

## Errors

| Sentinel | Status |
|---|---|
| `ErrSourceNotFound`, `ErrAnnotationNotFound` | 404 |
| `ErrInvalidSource`, `ErrInvalidAnnotation`, `ErrInvalidVerification`, `ErrInvalidChunk`, `ErrInvalidQuery` | 400 |
| `ErrUnverifiedCaller` | 401 |

`TestEverySentinelIsMapped` calls `Table.UnmappedSentinels`, so an exported
sentinel with no status fails the suite instead of arriving as a redacted 500.
