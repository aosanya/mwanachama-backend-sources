# Task board — done

| ID | Task | Done | Notes |
|---|---|---|---|
| SRC1 | Scaffold the module in the declared-domain shape: blueprint (source, annotation, verification, chunk), operations (13 addresses), `dispatch.Table` routes, MCP tools, `cmd/ddl`, wakala domain spec plus legal and newsroom examples, the guard tests | 2026-10-19 | `go test ./...` green. Guards proven to fail: removing the caller binding on `verify` reds two route tests, and a `court_approved` status reds the domain-word and vocabulary tests. |
