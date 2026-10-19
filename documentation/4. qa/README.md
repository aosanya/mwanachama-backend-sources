# QA

| Suite | Runs with | Holds |
|---|---|---|
| Root package tests | `go test ./...` (sqlite) | Manager behaviour per object, the verification identity rules, search ranking and model isolation |
| `routes/routes_test.go` | `go test ./...` | The declared table, every sentinel mapped, nothing anonymous, caller and address outranking the body over real HTTP |
| `domain_agnostic_test.go` | `go test ./...` | No domain word in an identifier, field, index or stored enum value |
| `examples_test.go` | `go test ./...` | Every shipped spec fits the Go types; three domains coexist in one database; no required field has a default |
| `vocabulary_test.go` | `go test ./...` | `models/vocabulary.go` and the blueprint's enum values agree in both directions |
| `postgres_integration_test.go` | `make test-pg` with `POSTGRES_URL` | jsonb path filtering on `properties`, embedding round trip, the 63-byte identifier limit |

The Postgres suite has not been run yet.
