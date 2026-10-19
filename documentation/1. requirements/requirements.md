# Requirements

## The ask (2026-10-19)

> Add backend-sources that is used to annotate the content. For instance a
> legal document needs this a lot. Consider RAG. It should have verified by.
> Can show name of who verified but also use the actor role. Should have
> custom properties like rights.

## What it must do

1. Hold **sources**: things content can be grounded in, described in the
   mounting domain's own words.
2. **Annotate** content held by *other* modules (a git blob, a forms answer, a
   catalog entry) with the source a passage rests on, the words relied on, and
   why.
3. Record **who verified** a source or an annotation, showing both their name
   and the actor role they held.
4. Carry **custom properties** per source, such as rights, licence and
   jurisdiction, without the module learning those words.
5. Support **retrieval-augmented generation**: hold source text as embedded
   passages and return the closest ones with their source and its
   verification attached.

## Decisions

| # | Decision | Why |
|---|---|---|
| 1 | Its own repo, not a feature of `git` | Annotations point into content owned by several modules. Inside `git`, every other module would depend on `git` just to cite something, and RAG storage belongs in neither. |
| 2 | Custom properties are one JSON document, `source.properties`, indexed by path per domain | The words (rights, jurisdiction, privilege) differ by domain. Catalog moved `Sector` into its document for the same reason. |
| 3 | `verified_by` comes from the caller; name and role are looked up, never read from the request | A body-supplied "verifier_role: Partner" would let anyone claim a partner's sign-off. |
| 4 | Name and role are snapshots taken at verification time | The record has to read correctly after a promotion or a name change. A live join would rewrite history. |
| 5 | Verifications are append-only and the newest is the current standing | A cite-check trail is evidence; the dispute that preceded a verification matters. Withdrawing is a new row with status `withdrawn`. |
| 6 | Without a profile lookup, only the actor id is recorded | Better an honest blank than a value the caller typed. |
| 7 | `source.kind` is free text | `statute`, `judgment` and `interview` are domain words. |
| 8 | Rights are descriptive, not enforced | Enforcement (blocking export, limiting readers) belongs to `permissions`. Filed as SRC5 rather than built speculatively. |
| 9 | Embeddings are stored as JSON arrays and compared in Go | `shared/spec` has no vector type. Correct now, slow at scale; pgvector is SRC4. |
| 10 | Search only compares chunks from the query's `embedding_model` | Vectors from different models are not comparable, and mixing them returns confident nonsense. |
| 11 | Deleting an annotation keeps its verifications | They are the record of what was checked. |
