# Resolved MCP tools

`mcp.RegisterTools(server, sm, mount)` binds every tool to one manager. A host
that holds one table set per tenant, as `mwanachama-wakala-api` does with one
per agency, cannot do that: an MCP server registers its tools once, at start,
and the manager depends on the call.

`mcp.RegisterResolvedTools(server, resolve, mount)` is the per-call shape,
following `mwanachama-backend-agency`'s `ManagerResolver`:

- Every tool gains a required `instance_id` argument. The handler takes it out
  of the arguments before `dispatch` sees them, so the declared tool schemas
  and the strict unknown-argument refusal are unchanged.
- The tools are built from `dispatch.Tools` over a delegating manager that
  calls `resolve(ctx, instance_id)` on every method, so the host's check
  (membership, in wakala-api) runs on every call and before any read or write.
- A missing `instance_id` is refused with `ErrInstanceRequired`. A resolver
  error is not one of `routes.Sentinels`, so `dispatch` answers it opaquely:
  a host's "not a member" and "no such instance" read the same to the caller.

`WithProfiles` is a manager option, so it is the resolver's to apply when it
builds a manager; nothing here changes it.
