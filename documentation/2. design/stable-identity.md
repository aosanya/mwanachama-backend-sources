# A thing's stable identity

`SourceManager` embeds the shared `identity.Owner` (shared S56), so a host that resolves a manager per instance already holds the owner for it. The `mcp` package's `resolving` manager forwards both methods to the manager it resolved for the call.

## The key

Nothing here is re-minted: a source, annotation, verification or chunk keeps its row id for as long as it exists, and `UpdateSource` does not touch it. So the stable key is the row id. The raw is

`sources/<instance>/<mount>/<object>/<id>`

for example `sources/agy1f2e3d4c/main/source/3f0c...`. The instance is the spec's instance. The mount and the object ride in the identity because a uuid does not say which of the four tables it is in, and because one instance can carry more than one mount of this module: a raw minted by one mount is `identity.ErrNotOwned` in another, not a confirmed miss that would mark a live thing stale.

`StableIdentity(ctx, object, id)` answers the raw for the four roles and `ErrNoStableIdentity` for anything else. `Resolve(ctx, raw)` answers the `(object, id)` back:

- `identity.ErrInvalidRaw` for a string that does not parse;
- `identity.ErrNotOwned` for another module's raw, another instance's or another mount's;
- `identity.ErrNotFound` for an identity that is not `<mount>/<object>/<id>`, an object this module lacks, or a row that is gone (a deleted source takes its chunks and verifications with it).

The ids are uuids, so asking `Resolve` about an id only confirms a row a caller already named exists. Nothing is read from the row.

A chunk is a passage `AddDocument` appended; its id is its key, and it has no other. If a document is ever re-split, the old chunks and their identities go.

## What the operations mark

`sourceID` and `annotationID` in `sources.operations.json` carry `refers` (`sources.source`, `sources.annotation`), the object name being the one `StableIdentity` takes. They are the only arguments the file binds one at a time. The annotation body's `source_id`, `target_module` and `target_id`, the filters' `source_id` and `target_id`, and a verification's `subject_id` arrive inside a whole body or a whole query, which the engine refuses to mark (S57), so they are unmarked until the engine can name a field of a whole request.

Tests: `operations_refers_test.go`.

## The conversation

An MCP tool call hands its `_meta` to the tool (`Tool.Handle`), so the chat's `conversation_id` reaches the manager on its context as `dispatch.ConversationOf(ctx)`, through both `RegisterTools` and `RegisterResolvedTools`. Tests: `mcp/tool_conversation_meta_test.go`.

Tests for identity: `identity_owner_test.go`.
