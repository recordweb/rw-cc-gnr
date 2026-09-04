// Command rw-cc-gnr startet den RecordWeb Global Namespace Registry
// Chaincode als eigenständigen Chaincode-Server (Fabric "external service"
// bzw. "chaincode-as-a-server" Betriebsmodus, je nach Peer-Konfiguration).
package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"

	ns "github.com/recordweb/rw-cc-gnr/namespaceregistry"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&ns.NamespaceContract{})
	if err != nil {
		log.Panicf("Fehler beim Erstellen des rw-cc-gnr Chaincodes: %v", err)
	}

	chaincode.Info.Title = "RecordWeb Global Namespace Registry"
	chaincode.Info.Version = "1.0.0"

	if err := chaincode.Start(); err != nil {
		log.Panicf("Fehler beim Starten des rw-cc-gnr Chaincodes: %v", err)
	}
}
