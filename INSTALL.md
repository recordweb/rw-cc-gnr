# Installing rw-cc-gnr as Chaincode-as-a-Service (CCaaS)

This guide documents the verified, working procedure for building and installing the `rw-cc-gnr` chaincode (RecordWeb Global Namespace Registry) as a Chaincode-as-a-Service (CCaaS) container on a Hyperledger Fabric channel operated by the `recordweb/rw-rrn` network repository.

It was written from a real first installation on the `rw-gnr-test` channel and includes every pitfall actually hit along the way. Follow it in order; each step depends on the previous one.

## Prerequisites

- The `rw-rrn` network is running (CA, orderers, peers, CLI container) via `docker compose` on the target VPS.
- The `rw-cc-gnr` repository is checked out on the VPS, e.g. at `/opt/rw-cc-gnr`, separate from `/opt/rw-rrn`.
- You have shell access to both directories and can run `docker compose exec cli ...` from `/opt/rw-rrn`.
- The channel you are installing on already exists (created via `configtxgen`/`osnadmin channel join`, out of scope for this document).

## Overview of the moving parts

Two repositories are involved, and they are **not** the same thing:

| Repository | Contains | Role |
|---|---|---|
| `rw-cc-gnr` | Go chaincode source, `Dockerfile` | Builds the chaincode server image |
| `rw-rrn` | `docker-compose.yml`, network config, CLI container | Runs the CCaaS container and all `peer`/`osnadmin` commands |

A CI/CD workflow in `rw-cc-gnr` can automate building the image and restarting the CCaaS container, but it **must not** automate the lifecycle steps (install/approve/commit) — those remain manual, console-driven actions on purpose, consistent with how CA bootstrap is handled in `rw-rrn` (crypto-material and chaincode-lifecycle changes are never a side effect of a routine `git push`).

## Step 0 — Naming convention used in this guide

Replace these placeholders with your actual values:

| Placeholder | Example used below | Meaning |
|---|---|---|
| `<CHANNEL>` | `rw-gnr-test` | Target Fabric channel |
| `<CC_NAME>` | `rw-cc-gnr` | Chaincode name on the channel |
| `<CC_VERSION>` | `1.0` | Chaincode version |
| `<CC_LABEL>` | `rw-cc-gnr_1.0` | Package label (`<CC_NAME>_<CC_VERSION>`) |
| `<CCAAS_CONTAINER>` | `ccaas-rw-cc-gnr` | Docker Compose service name for the chaincode server |
| `<ORG_MSP>` | `TWSOrgMSP` | Your peer organisation's MSP ID |
| `<ORDERER_HOST>` | `orderer0.tws.rwrrn.recordweb.dev:7050` | A reachable orderer endpoint |
| `<ORDERER_TLS_CA>` | path under `.../orderers/orderer0.../tls/ca.crt` | Orderer TLS CA cert |
| `<PEER1_ADDRESS>` | `peer1.tws.rwrrn.recordweb.dev:8051` | Second peer's address, if you run more than one |

For a production run, use e.g. `rw-gnr` / `rw-cc-gnr` / `rw-cc-gnr_1.0` / `ccaas-rw-cc-gnr` instead of the `-test` variants.

## Step 1 — Dockerfile (in `rw-cc-gnr`)

```dockerfile
# Using a newer Go toolchain than go.mod's minimum on purpose: the mock
# generator used below (counterfeiter) requires a newer Go version than
# go.mod declares as its minimum compatibility floor. Building with a
# newer toolchain is safe and does not change the module's declared
# minimum.
FROM golang:1.23 AS build
WORKDIR /chaincode

COPY . .

# namespaceregistry/mocks/ is intentionally NOT committed to the repo —
# it is generated test-double code (counterfeiter fakes of the Fabric
# interfaces). Pin counterfeiter to a version compatible with the Go
# version above; the newest release may require an even newer Go version
# than the image provides.
RUN go install github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2

# ORDER MATTERS. `go generate` must run BEFORE `go mod tidy`:
# mocks_gen.go and *_test.go files already import the not-yet-generated
# ".../mocks" package. If `go mod tidy` runs first, it cannot find that
# local package and tries (and fails) to resolve it as an external module
# ("no matching versions for query \"latest\"").
#
# `go generate` itself needs a resolvable module graph to typecheck the
# real Fabric interfaces it wraps. Plain `go mod download` populates the
# module cache but does NOT write a complete go.sum. `go mod tidy -e`
# tolerates the one currently-unresolvable import (the mocks package)
# while still writing correct checksums for everything else — enough for
# counterfeiter to succeed.
RUN go mod tidy -e
RUN go generate ./...

# Only now, with the mocks package physically present, does a full
# `go mod tidy` see a complete, resolvable import graph.
RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o /chaincode-server .

FROM debian:bookworm-slim
COPY --from=build /chaincode-server /chaincode-server
EXPOSE 9999
CMD ["/chaincode-server"]
```

**Why this specific structure, and not a simpler one:**

- No cached `go.mod`/`go.sum`-first layer trick is used. That pattern only works if `go.sum` is already complete and correct; if it was hand-edited or generated offline without real module-proxy access, it is more likely wrong than helpful, and a fresh, full resolution inside the build container (which has real internet access) is more reliable.
- `--lang` for `go mod`/`go generate` is irrelevant here; what matters is the *order* of `COPY . .` → `go mod tidy -e` → `go generate` → `go mod tidy` → `go build`.

## Step 2 — Add the CCaaS service to `docker-compose.yml` (in `rw-rrn`)

```yaml
  <CCAAS_CONTAINER>:
    image: rw-cc-gnr:<CC_VERSION>
    container_name: <CCAAS_CONTAINER>
    environment:
      - CHAINCODE_SERVER_ADDRESS=0.0.0.0:9999
      - CHAINCODE_ID=<CC_LABEL>:<PACKAGE_ID_HASH>   # filled in after Step 5
      - CORE_CHAINCODE_ID_NAME=<CC_LABEL>:<PACKAGE_ID_HASH>
    networks:
      - <same network as peer0/peer1>
    restart: unless-stopped
```

Commit this with a placeholder value for `<PACKAGE_ID_HASH>` first (or commit after Step 5 once you know the real hash). The container will not start correctly until the real package ID is in place, but that is expected at this stage.

## Step 3 — Build the image and start the container

If you have a CI/CD workflow that does this automatically on push, trigger it now. Otherwise, on the VPS:

```bash
cd /opt/rw-cc-gnr
docker build -t rw-cc-gnr:<CC_VERSION> .

cd /opt/rw-rrn
docker compose up -d --no-deps <CCAAS_CONTAINER>
docker compose logs <CCAAS_CONTAINER> --tail=30
```

Verify the container is actually running with the environment you expect:

```bash
docker compose ps <CCAAS_CONTAINER>
docker inspect <CCAAS_CONTAINER> --format '{{json .Config.Env}}'
```

## Step 4 — Create `connection.json` inside the CLI container

**Always run this from `/opt/rw-rrn`** (the directory containing the `docker-compose.yml` that defines the `cli` service). Running `docker compose exec cli ...` from `/opt/rw-cc-gnr` fails with `no configuration file provided: not found` — an easy mistake if you were just working in the chaincode repo's directory.

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
mkdir -p chaincode-packages/<CC_NAME>
cat > chaincode-packages/<CC_NAME>/connection.json <<EOF
{
  "address": "<CCAAS_CONTAINER>:9999",
  "dial_timeout": "10s",
  "tls_required": false
}
EOF
cat chaincode-packages/<CC_NAME>/connection.json
'
```

`tls_required: false` is acceptable here because traffic between the peer and the chaincode container stays inside the Docker-internal network and never leaves the host. Revisit this if that assumption changes.

## Step 5 — Package the chaincode

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
rm -rf chaincode-packages/<CC_NAME>
mkdir -p chaincode-packages/<CC_NAME>
cd chaincode-packages/<CC_NAME>

cat > connection.json <<EOF
{
  "address": "<CCAAS_CONTAINER>:9999",
  "dial_timeout": "10s",
  "tls_required": false
}
EOF

cat > metadata.json <<EOF
{
  "type": "ccaas",
  "label": "<CC_LABEL>"
}
EOF

tar cfz code.tar.gz connection.json
tar cfz <CC_NAME>.tar.gz metadata.json code.tar.gz

peer lifecycle chaincode package <CC_NAME>.tar.gz \
  --path . \
  --label <CC_LABEL>

echo "PACKAGE EXIT CODE: $?"
ls -la
'
```

**If you get `failed to marshal response: string field contains invalid UTF-8`** at the *install* step (not package), it is most likely caused by stale/partial archive files left over from a previous failed packaging attempt in the same directory. The fix is to `rm -rf` the whole package directory and rebuild `connection.json`, `metadata.json`, and both `.tar.gz` files from scratch in one clean run, as shown above (note the `rm -rf chaincode-packages/<CC_NAME>` at the top) — do not try to reuse or patch a directory that has already seen a failed package attempt.

## Step 6 — Install on every peer of your organisation

Peer0 (the `cli` container's default `CORE_PEER_*` target):

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
cd chaincode-packages/<CC_NAME>
peer lifecycle chaincode install <CC_NAME>.tar.gz
echo "INSTALL EXIT CODE (peer0): $?"
'
```

Every additional peer needs an explicit `CORE_PEER_*` override (adjust paths to your organisation's crypto material layout):

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
cd chaincode-packages/<CC_NAME>
CORE_PEER_ADDRESS=<PEER1_ADDRESS> \
CORE_PEER_TLS_CERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/<your-domain>/peers/<peer1-name>/tls/server.crt \
CORE_PEER_TLS_KEY_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/<your-domain>/peers/<peer1-name>/tls/server.key \
CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/<your-domain>/peers/<peer1-name>/tls/ca.crt \
peer lifecycle chaincode install <CC_NAME>.tar.gz
echo "INSTALL EXIT CODE (peer1): $?"
'
```

Both installs should report the **same** package identifier (it is a hash of the package content, independent of which peer installed it).

## Step 7 — Retrieve the package ID and wire it back into `docker-compose.yml`

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer lifecycle chaincode queryinstalled
'
```

Copy the reported identifier (format `<CC_LABEL>:<hash>`) into the `CHAINCODE_ID` and `CORE_CHAINCODE_ID_NAME` environment variables from Step 2, commit, and recreate the container so it actually runs with the correct ID:

```bash
cd /opt/rw-rrn
docker compose up -d --no-deps <CCAAS_CONTAINER>
docker inspect <CCAAS_CONTAINER> --format '{{json .Config.Env}}'
```

Confirm the printed environment actually contains the real hash, not a placeholder, before continuing.

## Step 8 — Approve for your organisation

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer lifecycle chaincode approveformyorg \
  -o <ORDERER_HOST> \
  --tls --cafile <ORDERER_TLS_CA> \
  --channelID <CHANNEL> \
  --name <CC_NAME> \
  --version <CC_VERSION> \
  --package-id <CC_LABEL>:<PACKAGE_ID_HASH> \
  --sequence 1
echo "APPROVE EXIT CODE: $?"
'
```

No `--signature-policy` flag is passed deliberately: omitting it makes the chaincode use the channel's default `Application/Endorsement` policy (typically `MAJORITY Endorsement` of the channel's members). This means "majority of organisations on the channel" is enforced automatically and recalculates itself as organisations join or leave — no manual endorsement policy maintenance is needed as the network grows. With a single organisation on the channel, "majority" trivially resolves to that one organisation; this does not need to be revisited later.

## Step 9 — Check commit readiness, then commit

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer lifecycle chaincode checkcommitreadiness \
  --channelID <CHANNEL> \
  --name <CC_NAME> \
  --version <CC_VERSION> \
  --sequence 1 \
  --output json
'
```

Confirm the output shows `"<ORG_MSP>": true` (and `true` for every other required organisation, if there is more than one) before proceeding:

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer lifecycle chaincode commit \
  -o <ORDERER_HOST> \
  --tls --cafile <ORDERER_TLS_CA> \
  --channelID <CHANNEL> \
  --name <CC_NAME> \
  --version <CC_VERSION> \
  --sequence 1
echo "COMMIT EXIT CODE: $?"
'
```

Verify:

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer lifecycle chaincode querycommitted --channelID <CHANNEL> --name <CC_NAME>
'
```

## Step 10 — Functional smoke test

```bash
cd /opt/rw-rrn
docker compose exec cli bash -c '
peer chaincode invoke \
  -o <ORDERER_HOST> \
  --tls --cafile <ORDERER_TLS_CA> \
  -C <CHANNEL> \
  -n <CC_NAME> \
  -c "{\"function\":\"RegisterNamespace\",\"Args\":[\"https://example.org/resolver/1.0/identifiers\"]}"
'
```

`RegisterNamespace` takes only a `resolverEndpoint` argument — it does not accept a caller-supplied namespace. The chaincode derives the namespace deterministically from the transaction ID and returns it in the resulting record. Copy the returned `namespace` value and confirm both read paths:

```bash
docker compose exec cli bash -c '
peer chaincode query -C <CHANNEL> -n <CC_NAME> \
  -c "{\"function\":\"ResolveNamespace\",\"Args\":[\"<NAMESPACE_FROM_ABOVE>\"]}"
'

docker compose exec cli bash -c '
peer chaincode query -C <CHANNEL> -n <CC_NAME> \
  -c "{\"function\":\"GetMyNamespaces\",\"Args\":[]}"
'
```

`GetMyNamespaces` takes no arguments and returns only namespaces registered by the calling organisation's authenticated identity — this is by design (see the chaincode's own README for the rationale).

## Troubleshooting quick reference

| Symptom | Cause | Fix |
|---|---|---|
| `no configuration file provided: not found` | Ran `docker compose exec` from the wrong directory | `cd` into the directory holding `docker-compose.yml` (the network repo, not the chaincode repo) |
| `unknown chaincodeType: EXTERNAL` | `metadata.json` used `"type": "external"` | Use `"type": "ccaas"` |
| `failed to determine module root: exec: "go": executable file not found` | `--lang` omitted or set to `golang`/`external` for a CCaaS package | Use `--lang node` |
| `failed to marshal response: string field contains invalid UTF-8` (at install) | Stale/partial files from a previous failed package attempt | `rm -rf` the package directory and rebuild everything from scratch |
| `no such service: <name>` (compose) | Service not yet defined/committed in `docker-compose.yml` on this host | Commit and pull the compose change before restarting the service |
| `go: ... no matching versions for query "latest"` (in Docker build) | `go mod tidy` ran before `go generate` created the local `mocks` package | Reorder: `go mod tidy -e` → `go generate ./...` → `go mod tidy` |
| `go: ... requires go >= 1.2X` (installing a Go dev tool in the build stage) | Build image's Go version is older than the tool's minimum | Bump the Dockerfile's base image tag, and/or pin the tool to an older compatible release instead of `@latest` |
| `invalid version: unknown revision ...` (in `go.mod`) | A hand-written pseudo-version does not actually exist | Do not hand-write dependency pseudo-versions; let `go mod tidy` resolve them inside a container with real network access |

## Notes for a production (non-test) channel

- Re-run the full sequence above end-to-end against the production channel; nothing can be skipped or assumed identical purely because the test channel succeeded — package IDs, in particular, will differ.
- Record the successful installation (date, chaincode version, sequence, package ID) in the network's operating/build log, since this is a governance-relevant milestone, not just a technical one.
