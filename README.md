# mwanachama-backend-sources

Grounds content in the sources it rests on. A **source** is something a
reader can be pointed at — a statute, a ruling, a standard, a dataset. An
**annotation** pins a passage of content held by another module to a source.
A **verification** records who stood behind a source or an annotation, by
name *and* by the role they held at the time. **Chunks** hold a source's text
as passages with embeddings, so a drafting assistant retrieves and cites a
real source rather than inventing one.

A Go library in the declared-domain shape: objects in
[sources.blueprint.json](sources.blueprint.json), the route table in
[sources.operations.json](sources.operations.json), storage through
`mwanachama-backend-shared/specstore`, routes and MCP tools through
`mwanachama-backend-shared/dispatch`. No server of its own.

```go
s, _ := sources.SpecFor("app")
_ = sources.Provision(db, s)
sm, _ := sources.NewSourceManager(db, s, sources.WithProfiles(lookup))
for _, rt := range routes.RoutesFor(sm, routes.Mount{Caller: callerID}) {
	mux.HandleFunc(rt.Pattern("/sources-svc"), rt.Handler)
}
```

```sh
make test      # sqlite, no database needed
make test-pg   # POSTGRES_URL=... jsonb path indexes and identifier limits
go run ./cmd/ddl spec/examples/legal.sources.json
```

See [documentation/](documentation/).
