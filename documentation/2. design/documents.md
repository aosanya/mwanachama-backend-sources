# Adding a document

HS31. `POST /sources/{sourceID}/documents` (`AddDocument`, action
`sources.document.create`) takes a source's whole text and files it as
chunks, so an answer built from a passage can cite the section and the exact
words it came from. No embeddings are produced; a chunk filed this way has an
empty `embedding` and `embedding_model` until HS32 supplies them.

## The body

| Field | Meaning |
| --- | --- |
| `text` | The whole document. Required, and it must hold more than whitespace. |
| `format` | `text` (the default) or `markdown`. Anything else is `ErrInvalidDocument`, 400. |
| `source_id` | Ignored: the address decides, the same as on `add_chunk`. |

The response is the chunks filed, in order.

## How it splits

- A passage is a run of non-blank lines. A blank line ends it.
- In `markdown`, an ATX heading (`#` to `######`, followed by a space or the
  end of the line) also ends the passage before it, and is not itself a
  passage. Lines inside a ``` fence are never headings and blank lines inside
  one do not split.
- In `text`, a line starting with `#` is ordinary text.
- A document that leaves no passages, only headings, is refused.

## What each chunk keeps

- `heading` — the headings the passage sits under, outermost first, joined by
  ` / ` (`Opening hours / Weekends`). A heading clears every deeper one, and a
  skipped level is simply absent from the path. Empty under no heading and
  for a chunk filed on its own through `add_chunk`.
- `start_offset`, `end_offset` — the passage's byte range in the submitted
  text, end exclusive, so `text[start_offset:end_offset]` is the chunk's text
  exactly. Leading and trailing spaces and tabs of the passage are outside the
  range. They are bytes, not characters: a client slicing in UTF-16 has to
  convert. A chunk filed on its own carries zeros, and `add_chunk` refuses a
  negative start or an end before the start.
- `position` — continues after the chunks the source already holds, so a
  second document on the same source reads back after the first.

## Why one transaction

All of a document's chunks are written in one transaction, so a failure part
way leaves the source with none of them rather than half a document whose
positions have a gap.

## Why a separate action

`sources.document.create` is its own action rather than reusing
`sources.chunk.create`, because the dispatcher refuses two operations claiming
one action. A grant table that allows filing chunks needs this action added
to allow filing documents.

## Not here

Upload, PDF or other file text extraction, embeddings (HS32), and mounting in
wakala-api are other rows.
