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

`namespace` is always a **canonical, lowercase UUIDv4**-formatted string
(RWP #23). `resolverEndpoint` must be an absolute **HTTPS** URL.

## Namespace Assignment Model

**A namespace is never supplied by the caller.** `RegisterNamespace` takes
only `resolverEndpoint` as input; the namespace value itself is derived
deterministically inside the chaincode from the current transaction ID
(SHA-256 hash of `GetTxID()`, reshaped into RFC 4122 UUIDv4 byte layout —
see `namespace_id.go`).

This is a deliberate design decision, not an oversight:

- **A namespace may only ever be registered once and therefore have
  exactly one `resolverEndpoint`.** This is the collision the derivation
  scheme protects against: a caller-chosen namespace string carries no
  uniqueness guarantee beyond the caller's own care, whereas a
  transaction-ID-derived value is guaranteed unique as long as at most one
  namespace is derived per transaction.
- **The same `resolverEndpoint` may be reused across many separate
  `RegisterNamespace` calls, and this is intentional.** Each call yields
  its own, newly derived, distinct namespace pointing at that endpoint.
  It is up to the calling organisation to track which of its namespaces
  is used for which purpose — the registry does not enforce or track that
  intent.

**Why not a "real" random UUIDv4 (`uuid.New()`)?** Chaincode must produce
byte-identical results on every endorsing peer for a transaction to pass
endorsement. True randomness would yield a different value on each peer
and break endorsement comparison as soon as more than one organisation
endorses — which is always the case on this network (multi-org
endorsement policy, see `rw-rrn`). Fabric's transaction ID is itself a
SHA-256 hash computed once by the submitting client and included in the
transaction envelope, so it is identical across all endorsing peers.
Hashing it again only reshapes it into the UUIDv4 byte layout; the
resulting string is syntactically indistinguishable from a randomly
generated UUIDv4, it simply is not drawn from a random source — an
accepted, deliberate trade-off for chaincode determinism.

## Public Transaction Functions

| Function | Type | Description |
|---|---|---|
| `RegisterNamespace(resolverEndpoint)` | Submit | Registers a new namespace for the given endpoint and returns the full resulting record (including the newly derived `namespace`). Neither `namespace` nor `registeredBy` is a caller-supplied argument. |
| `UpdateResolverEndpoint(namespace, newResolverEndpoint)` | Submit | Updates the resolver endpoint of an existing namespace. Only the organisation that originally registered the namespace may do this. |
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
of polling the ledger. This is particularly relevant for
`RegisterNamespace`, since the caller no longer chooses the namespace up
front — the event (and the direct return value) are the two ways to learn
which namespace was assigned.

## Determinism

`RegisteredAt`/`UpdatedAt` are derived from `ctx.GetStub().GetTxTimestamp()`,
**not** from `time.Now()`. `time.Now()` returns a slightly different value
on each endorsing peer and would cause an endorsement mismatch — a known
anti-pattern in Fabric chaincode. The same determinism requirement is why
the namespace itself is derived from `GetTxID()` rather than generated
with a random UUID library (see "Namespace Assignment Model" above).

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
│   ├── namespace_id.go              # Deterministic namespace derivation from TxID
│   ├── time_util.go                 # Deterministic timestamp formatting
│   ├── register.go                  # RegisterNamespace, UpdateResolverEndpoint
│   ├── resolve.go                   # ResolveNamespace, GetMyNamespaces
│   ├── history.go                   # GetNamespaceHistory
│   ├── mocks_gen.go                 # go:generate directives for counterfeiter
│   ├── mocks/                       # generated mocks (after go generate)
│   ├── register_test.go
│   ├── resolve_test.go
│   ├── history_test.go
│   ├── namespace_id_test.go
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
- **Namespace derivation and RWP #23**: the derived namespace is
  syntactically a canonical UUIDv4 (RFC 4122 version 4, variant 10) but is
  not drawn from a random source — it is deterministically derived from
  the transaction ID for chaincode-determinism reasons (see "Namespace
  Assignment Model"). If RWP #23 is ever clarified to require genuine
  entropy rather than UUIDv4 *syntax*, this derivation scheme would need
  to be revisited.

