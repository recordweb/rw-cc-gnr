package namespaceregistry

// NamespaceRecord ist die on-chain Datenstruktur für einen RWP Global
// Namespace Identifier (canonical UUIDv4, gemäss RWP #23).
//
// Der Registry-Chaincode speichert ausschliesslich Routing-Metadaten.
// Er darf gemäss RWP #25 NIEMALS DID-Dokumente, Records, Record-Inhalte
// oder Access-Control-Entscheidungen enthalten.
type NamespaceRecord struct {
	// DocType erlaubt künftig mehrere Key-Präfixe/Datentypen im selben
	// World State zu unterscheiden (z.B. für spätere CouchDB Rich Queries
	// via Index auf "docType"). Aktuell immer "namespaceRecord".
	DocType string `json:"docType"`

	// Namespace ist der canonical UUIDv4 Global Namespace Identifier.
	// Dies ist zugleich der Ledger-Key.
	Namespace string `json:"namespace"`

	// ResolverEndpoint ist die HTTPS-URL, unter der der Namespace aufgelöst
	// werden kann. Muss gemäss Validierung ein absolutes https://-URL sein.
	ResolverEndpoint string `json:"resolverEndpoint"`

	// RegisteredBy ist die MSP-ID der registrierenden Organisation.
	// Wird IMMER aus der authentifizierten Fabric-Client-Identität
	// abgeleitet (ctx.GetClientIdentity().GetMSPID()) und ist niemals
	// per Transaktionsargument überschreibbar (RWP #25, Kommentar).
	RegisteredBy string `json:"registeredBy"`

	// RegisteredAt ist der deterministische Transaktions-Zeitstempel
	// (ctx.GetStub().GetTxTimestamp()), NICHT time.Now(). time.Now() ist
	// in Chaincode nicht deterministisch zwischen Peers und würde bei
	// der Endorsement-Validierung zu Result-Mismatches führen.
	RegisteredAt string `json:"registeredAt"`

	// UpdatedAt wird bei jeder Mutation aktualisiert; bei Erstregistrierung
	// identisch mit RegisteredAt.
	UpdatedAt string `json:"updatedAt"`

	// TxID ist die Transaktions-ID der letzten Mutation dieses Records.
	TxID string `json:"txId"`

	// EndorsedBy listet die MSP-IDs, deren Endorsement für die jeweilige
	// Transaktion tatsächlich vorlag (informativ; die verbindliche Prüfung
	// erfolgt durch die Channel-Endorsement-Policy, nicht durch dieses Feld).
	EndorsedBy []string `json:"endorsedBy"`

	// SchemaVersion erlaubt künftige, abwärtskompatible Erweiterungen der
	// Datenstruktur ohne bestehende Records ungültig zu machen.
	SchemaVersion int `json:"schemaVersion"`
}

// CurrentSchemaVersion wird bei jeder Struktur-Änderung von NamespaceRecord
// erhöht. Lesefunktionen sollten (perspektivisch) alte Versionen erkennen
// und ggf. auf Best-Effort-Basis migrieren/kennzeichnen können.
const CurrentSchemaVersion = 1

// HistoryEntry wrappt einen einzelnen Ledger-History-Eintrag mit
// Transaktions-Metadaten für GetNamespaceHistory.
type HistoryEntry struct {
	TxID      string           `json:"txId"`
	Timestamp string           `json:"timestamp"`
	IsDelete  bool             `json:"isDelete"`
	Record    *NamespaceRecord `json:"record,omitempty"`
}

// docTypeNamespaceRecord ist der feste DocType-Wert für NamespaceRecord-Keys.
const docTypeNamespaceRecord = "namespaceRecord"
