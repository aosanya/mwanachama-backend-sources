# Objects and domains

[`sources.blueprint.json`](../../sources.blueprint.json) declares four roles.
A domain spec names them and chooses their tables.

| Role | wakala (`sources.wakala.json`) | legal (example) | newsroom (example) |
|---|---|---|---|
| `source` | source | authority | record |
| `annotation` | annotation | citation | attribution |
| `verification` | verification | cite_check | fact_check |
| `chunk` | chunk | passage | excerpt |

The two examples exist so that domain neutrality is actually exercised:
`TestTwoDomainsCoexist` migrates all three into one database. Neither example
is production config.

## Custom properties

`source.properties` is a JSON document. The legal example adds path indexes:

```json
{ "name": "rights", "path": { "field": "properties", "path": "rights" } }
```

In Postgres that becomes
`create index … ((properties #>> '{rights}'))`. Run
`go run ./cmd/ddl spec/examples/legal.sources.json` to read the DDL.

## How an annotation points into content

`target_module` + `target_id` name the item, as the host names its modules and
the module names its items. `anchor` is a JSON document because every holding
module points inside its content differently: a character range in a git
blob, a question id in a form, a heading path in a catalog document. `quote`
copies the source's words so a reader can check them without opening it.

## Retrieval

`SearchChunks(ChunkQuery{Embedding, EmbeddingModel, SourceID?, Limit?})`:

1. Load the chunks with the same `embedding_model`, optionally limited to one
   source.
2. Score by cosine similarity. A chunk whose vector has a different width is
   skipped, not compared.
3. Return the top `Limit` (default 5, max 50) as `ChunkMatch{Chunk, Source,
   Standing, Score}`, where `Standing` is the source's newest verification.

The module doesn't produce embeddings. Whoever files a chunk brings its
vector, and whoever searches brings the query's vector from the same model.
