# rw-cc-gnr — RecordWeb Global Namespace Registry Chaincode

Hyperledger Fabric Chaincode für die **RecordWeb Global Namespace Registry** (RW-GNR).
Läuft auf dem Fabric-Channel `rw-gnr` (Produktion) bzw. `rw-gnr-test` (Testnetz),
betrieben im Netzwerk-Repo [`recordweb/rw-rrn`](https://github.com/recordweb/rw-rrn).

## Zweck und Grenzen

Der Chaincode implementiert die Namespace-Registry gemäss RWP-Konzept, Kapitel 12.2,
und den normativen Anforderungen aus:

- RWC [#17](https://github.com/recordweb/rwc/issues/17) — Governance/Betriebsanforderungen
- RWP [#23](https://github.com/recordweb/rwp/issues/23) — `did:rwp`-Syntax, canonical UUIDv4
- RWP [#24](https://github.com/recordweb/rwp/issues/24) — Globales Namespace-Resolution-Modell
- RWP [#25](https://github.com/recordweb/rwp/issues/25) — Hyperledger Fabric Profil

**Was die Registry speichert:** ausschliesslich Routing-Metadaten
(`namespace`, `resolverEndpoint`, `registeredBy`, `registeredAt`, `txId`,
plus `updatedAt`, `endorsedBy`, `schemaVersion`).

**Was die Registry NICHT speichert (RWP #25):** DID-Dokumente, Records,
Record-Inhalte, Access-Control-Entscheidungen. Diese Grenze ist bewusst
und darf durch künftige Erweiterungen nicht aufgeweicht werden.

## Datenstruktur

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

`namespace` muss ein **kanonischer, kleingeschriebener UUIDv4** sein (RWP #23).
`resolverEndpoint` muss eine absolute **HTTPS**-URL sein.

## Öffentliche Transaktionsfunktionen

| Funktion | Typ | Beschreibung |
|---|---|---|
| `RegisterNamespace(namespace, resolverEndpoint)` | Submit | Registriert einen neuen Namespace. `registeredBy` wird aus der authentifizierten Client-Identität abgeleitet, nicht aus einem Argument. |
| `UpdateResolverEndpoint(namespace, newResolverEndpoint)` | Submit | Aktualisiert den Resolver-Endpoint. Nur die ursprünglich registrierende Organisation darf dies tun. |
| `ResolveNamespace(namespace)` | Evaluate | Liefert den vollständigen Record für einen Namespace. Öffentlich lesbar für alle Channel-Teilnehmer. |
| `GetMyNamespaces()` | Evaluate | Liefert alle Namespaces, die von der aufrufenden Organisation registriert wurden. Kein Parameter — die Identität kommt ausschliesslich aus dem Client-Kontext. |
| `GetNamespaceHistory(namespace)` | Evaluate | Liefert die vollständige, unveränderliche Änderungshistorie eines Namespace. |

**Bewusst nicht enthalten:** Ein `GetAllNamespaces` über die gesamte Registry
wurde verworfen (Skalierungsrisiko bei unbeschränktem Full-Scan, kein
Bestandteil der normativen Mindest-API). Ein Admin-/Audit-Bedürfnis über
alle Organisationen hinweg sollte extern gelöst werden (z. B. Block-/
State-Listener, der eine Kopie in eine externe DB spiegelt), nicht durch
eine Chaincode-Funktion, die für Reporting-Workloads nicht vorgesehen ist.

## Identität und Autorisierung

Die Registrar-Identität wird **immer** aus `ctx.GetClientIdentity().GetMSPID()`
abgeleitet, niemals aus einem Transaktionsargument (RWP #25, Kommentar).
Ein Client kann sich also nicht als andere Organisation ausgeben, auch nicht
versehentlich durch einen falschen Parameter.

`UpdateResolverEndpoint` prüft, dass die aufrufende MSP-ID mit `registeredBy`
des existierenden Records übereinstimmt. Ein Update durch eine andere
Organisation schlägt mit `UNAUTHORIZED_REGISTRAR` fehl.

Die eigentliche Durchsetzung von Mehrorganisations-Konsens erfolgt über die
**Channel-Endorsement-Policy** (ausserhalb dieses Chaincodes, siehe
`rw-rrn`-Repo), nicht durch Logik im Chaincode selbst.

## Fehlerbehandlung

Alle Fehler sind vom Typ `ContractError` mit einem stabilen `ErrorCode`
(siehe `errors.go`), z. B. `INVALID_NAMESPACE_FORMAT`, `NAMESPACE_ALREADY_EXISTS`,
`UNAUTHORIZED_REGISTRAR`, `LEDGER_READ_FAILED`. Aufrufende Clients (Admin-GUI,
Resolver-Dienste) können anhand des Codes programmatisch reagieren, statt
Freitext-Fehlermeldungen zu parsen.

## Logging

Strukturierte JSON-Logs nach stdout (`INFO`/`WARN`) bzw. stderr (`ERROR`),
siehe `logging.go`. Jede Log-Zeile enthält Funktion, Transaktions-ID, MSP-ID
und Namespace, sofern vorhanden. Peers leiten Chaincode-Container-stdout/stderr
in ihre eigenen Logs weiter; das JSON-Format erlaubt späteres Einsammeln
durch Log-Aggregatoren (Loki/ELK), ohne Freitext parsen zu müssen.

## Events

- `NamespaceRegistered` — bei jeder erfolgreichen `RegisterNamespace`-Transaktion.
- `NamespaceUpdated` — bei jeder erfolgreichen `UpdateResolverEndpoint`-Transaktion.

Beide Events tragen den vollständigen, serialisierten `NamespaceRecord` als Payload.
Externe Konsumenten (z. B. ein Resolver-Cache) können sich darauf abonnieren,
statt den Ledger zu pollen.

## Determinismus

`RegisteredAt`/`UpdatedAt` werden aus `ctx.GetStub().GetTxTimestamp()` abgeleitet,
**nicht** aus `time.Now()`. `time.Now()` liefert auf jedem endorsierenden Peer
einen leicht unterschiedlichen Wert und würde zu einem Endorsement-Mismatch
führen — ein bekanntes Antipattern in Fabric-Chaincode.

## State-Database-Kompatibilität

Der Chaincode ist so geschrieben, dass er sowohl mit LevelDB als auch mit
CouchDB als Peer-State-Database funktioniert (reine Key/Value- bzw.
Key-Range-Zugriffe, kein CouchDB-spezifisches Rich-Query im aktuellen Stand).
`GetMyNamespaces` filtert aktuell in-memory nach einem vollständigen
Range-Scan; bei CouchDB als State-Database (siehe Migration im `rw-rrn`-Repo)
kann dies künftig durch eine indizierte Mango-Query auf `registeredBy`
ersetzt werden, was bei wachsender Registry-Grösse deutlich günstiger ist.
Das `docType`-Feld ist bereits für einen künftigen CouchDB-Index vorgesehen.

## Entwicklung

```bash
go mod tidy

# Mocks generieren (einmalig counterfeiter installieren)
go install github.com/maxbrunsfeld/counterfeiter/v6@latest
go generate ./...

# Tests ausführen
go test ./... -v

# Build prüfen
go build ./...
```

## Verzeichnisstruktur

```
rw-cc-gnr/
├── go.mod
├── main.go                          # Chaincode-Server-Einstiegspunkt
├── namespaceregistry/
│   ├── contract.go                  # SmartContract-Struct + Package-Dokumentation
│   ├── types.go                     # NamespaceRecord, HistoryEntry
│   ├── errors.go                    # ContractError, ErrorCode-Konstanten
│   ├── logging.go                   # Strukturiertes JSON-Logging
│   ├── validation.go                # UUIDv4- und HTTPS-URL-Validierung
│   ├── identity.go                  # Ableitung der Registrar-MSP-ID
│   ├── time_util.go                 # Deterministische Zeitstempel-Formatierung
│   ├── register.go                  # RegisterNamespace, UpdateResolverEndpoint
│   ├── resolve.go                   # ResolveNamespace, GetMyNamespaces
│   ├── history.go                   # GetNamespaceHistory
│   ├── mocks_gen.go                 # go:generate-Direktiven für counterfeiter
│   ├── mocks/                       # generierte Mocks (nach go generate)
│   ├── register_test.go
│   ├── resolve_test.go
│   ├── history_test.go
│   └── validation_test.go
└── README.md
```

## Offene Punkte / bekannte Grenzen

- **Pagination**: `GetMyNamespaces` ist für die aktuelle Registry-Grösse
  (wenige hundert Namespaces) unproblematisch, aber nicht paginiert.
  Sollte die Anzahl der Namespaces pro Organisation stark wachsen, ist eine
  `GetMyNamespacesPaginated(pageSize, bookmark)`-Variante mit
  `GetStateByRangeWithPagination` sinnvoll.
- **CouchDB-Indizes**: Sobald das Netzwerk auf CouchDB migriert ist
  (siehe `rw-rrn`), sollte ein `_design`-Dokument mit Index auf
  `docType` + `registeredBy` ergänzt werden, um `GetMyNamespaces` von
  einem Full-Scan auf eine indizierte Query umzustellen.
- **Schema-Migration**: `schemaVersion` ist vorbereitet, aber es existiert
  noch keine Migrationslogik für bestehende Records bei künftigen
  Strukturänderungen.
- **Integrationstests**: Diese Unit-Tests laufen gegen generierte Mocks,
  nicht gegen ein echtes Fabric-Netzwerk. Ein Integrationstest gegen
  `rw-gnr-test` (siehe `rw-rrn`-Repo) ist vor jedem Produktions-Deploy
  weiterhin nötig.
