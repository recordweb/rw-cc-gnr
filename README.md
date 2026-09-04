# rw-cc-gnr — RecordWeb Global Namespace Registry Chaincode

Hyperledger Fabric chaincode for the **RecordWeb Global Namespace Registry** (RW-GNR).
Runs on the Fabric channel `rw-gnr` (production) resp. `rw-gnr-test` (testnet),
operated in the network repository [`recordweb/rw-rrn`](https://github.com/recordweb/rw-rrn).

## Purpose and Boundaries

The chaincode implements the namespace registry as specified in the RWP
concept, chapter 12.2, and the normative requirements from:

- RWC [#17](https://github.com/recordweb/rwc/issues/17) — governance/operating requirements
- RWP [#23](https://github.com/recordweb/rwp/issues/23) — `did:rwp` syntax, canonical UUIDv4
- RWP [#24](https://github.com/recordweb/rwp/issues/24) — global namespace-resolution model
- RWP [#25](https://github.com/recordweb/rwp/issues/25) — Hyperledger Fabric profile

**What the registry stores:** routing metadata only
(`namespace`, `resolverEndpoint`, `registeredBy`, `registeredAt`, `txId`,
plus `updatedAt`, `endorsedBy`, `schemaVersion`).

**What the registry does NOT store (RWP #25):** DID documents, Records,
Record content, or access-control decisions. This boundary is deliberate
and must not be weakened by future extensions of this chaincode.

## Data Structure

```json
{
  "docType": "namespaceRecord",
  "namespace": "a3f9e21c-1234-4abc-8def-1234567890ab",
  "resolverEndpoint": "https://vps.recordweb.dev/resolver/parlament/1.0/identifiers",
  "registeredBy": "SwissGovOrgMSP",
  "registeredAt": "2026-07-23T15:10:50Z",
  "updatedAt": "2026-07-23T15:10:50Z",
  "txId": "a1b2c3d4...",
  "endorsedBy": ["SwissGovOrgMSP"],
  "schemaVersion": 1
}
```

`namespace` must be a **canonical, lowercase UUIDv4** (RWP #23).
`resolverEndpoint` must be an absolute **HTTPS** URL.

## Public Transaction Functions

| Function | Type | Description |
|---|---|---|
| `RegisterNamespace(namespace, resolverEndpoint)` | Submit | Registers a new namespace. `registeredBy` is derived from the authenticated client identity, never from an argument. |
| `UpdateResolverEndpoint(namespace, newResolverEndpoint)` | Submit | Updates the resolver endpoint. Only the organisation that originally registered the namespace may do this. |
| `ResolveNamespace(namespace)` | Evaluate | Returns the full record for a namespace. Publicly readable by all channel members. |
| `GetMyNamespaces()` | Evaluate | Returns all namespaces registered by the calling organisation. No parameter — the identity comes exclusively from the client context. |
| `GetNamespaceHistory(namespace)` | Evaluate | Returns the full, immutable modification history of a namespace. |

**Deliberately not included:** A `GetAllNamespaces` function across the
entire registry was dropped (scalability risk from an unbounded full scan,
and not part of the normative minimum API). An admin/audit need spanning
all organisations should be solved externally (e.g. a block/state listener
mirroring a copy into an external database), not through a chaincode
function that isn't designed for reporting workloads.

## Identity and Authorisation

The registrar identity is **always** derived from
`ctx.GetClientIdentity().GetMSPID()`, never from a transaction argument
(RWP #25, comment). A client can therefore never impersonate another
organisation, not even by accident through a wrong parameter.

`UpdateResolverEndpoint` checks that the calling MSP ID matches
`registeredBy` on the existing record. An update attempted by a different
organisation fails with `UNAUTHORIZED_REGISTRAR`.

Actual multi-organisation consensus enforcement happens through the
**channel endorsement policy** (outside this chaincode, see the `rw-rrn`
repository), not through logic inside this chaincode.

## Error Handling

All errors are of type `ContractError` with a stable `ErrorCode` (see
`errors.go`), e.g. `INVALID_NAMESPACE_FORMAT`, `NAMESPACE_ALREADY_EXISTS`,
`UNAUTHORIZED_REGISTRAR`, `LEDGER_READ_FAILED`. Calling clients (admin GUI,
resolver services) can react programmatically based on the code instead of
parsing free-text error messages.

## Logging

Structured JSON logs to stdout (`INFO`/`WARN`) resp. stderr (`ERROR`), see
`logging.go`. Every log line includes the function name, transaction ID,
MSP ID, and namespace, where applicable. Peers forward chaincode container
stdout/stderr into their own logs; the JSON format allows later ingestion
by log aggregators (Loki/ELK) without having to parse free text.

## Events

- `NamespaceRegistered` — emitted on every successful `RegisterNamespace` transaction.
- `NamespaceUpdated` — emitted on every successful `UpdateResolverEndpoint` transaction.

Both events carry the full, serialized `NamespaceRecord` as payload.
External consumers (e.g. a resolver cache) can subscribe to these instead
of polling the ledger.

## Determinism

`RegisteredAt`/`UpdatedAt` are derived from `ctx.GetStub().GetTxTimestamp()`,
**not** from `time.Now()`. `time.Now()` returns a slightly different value
on each endorsing peer and would cause an endorsement mismatch — a known
anti-pattern in Fabric chaincode.

## State Database Compatibility

The chaincode is written to work with both LevelDB and CouchDB as the peer
state database (plain key/key-range access only, no CouchDB-specific rich
query in the current state). `GetMyNamespaces` currently filters in-memory
after a full range scan; once the network migrates to CouchDB (see the
`rw-rrn` repository), this can be replaced by an indexed Mango query on
`registeredBy`, which is significantly cheaper as the registry grows. The
`docType` field is already in place to support a future CouchDB index.

## Development

```bash
go mod tidy

# Generate mocks (install counterfeiter once)
go install github.com/maxbrunsfeld/counterfeiter/v6@latest
go generate ./...

# Run tests
go test ./... -v

# Verify build
go build ./...
```

## Directory Structure

```
rw-cc-gnr/
├── go.mod
├── main.go                          # Chaincode server entry point
├── namespaceregistry/
│   ├── contract.go                  # SmartContract struct + package documentation
│   ├── types.go                     # NamespaceRecord, HistoryEntry
│   ├── errors.go                    # ContractError, ErrorCode constants
│   ├── logging.go                   # Structured JSON logging
│   ├── validation.go                # UUIDv4 and HTTPS URL validation
│   ├── identity.go                  # Registrar MSP ID derivation
│   ├── time_util.go                 # Deterministic timestamp formatting
│   ├── register.go                  # RegisterNamespace, UpdateResolverEndpoint
│   ├── resolve.go                   # ResolveNamespace, GetMyNamespaces
│   ├── history.go                   # GetNamespaceHistory
│   ├── mocks_gen.go                 # go:generate directives for counterfeiter
│   ├── mocks/                       # generated mocks (after go generate)
│   ├── register_test.go
│   ├── resolve_test.go
│   ├── history_test.go
│   └── validation_test.go
└── README.md
```

## Open Items / Known Limitations

- **Pagination**: `GetMyNamespaces` is fine for the current registry size
  (a few hundred namespaces) but is not paginated. If the number of
  namespaces per organisation grows significantly, a
  `GetMyNamespacesPaginated(pageSize, bookmark)` variant using
  `GetStateByRangeWithPagination` would be advisable.
- **CouchDB indexes**: Once the network has migrated to CouchDB (see
  `rw-rrn`), a `_design` document indexing `docType` + `registeredBy`
  should be added to move `GetMyNamespaces` from a full scan to an
  indexed query.
- **Schema migration**: `schemaVersion` is in place, but no migration
  logic exists yet for existing records under future structural changes.
- **Integration tests**: These unit tests run against generated mocks,
  not against a real Fabric network. An integration test against
  `rw-gnr-test` (see the `rw-rrn` repository) is still required before
  every production deployment.
