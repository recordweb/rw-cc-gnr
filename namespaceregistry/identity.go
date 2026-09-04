package namespaceregistry

import "github.com/hyperledger/fabric-contract-api-go/contractapi"

// callerMSPID liefert die MSP-ID des aufrufenden Clients, abgeleitet aus
// dessen authentifiziertem X.509-Zertifikat. Dies ist die EINZIGE zulässige
// Quelle für "wer registriert/aktualisiert diesen Namespace" — niemals ein
// Transaktionsargument (RWP #25, Kommentar: "Registrar identity must derive
// from the authenticated Fabric client identity and must not be overridable
// via transaction arguments").
func callerMSPID(ctx contractapi.TransactionContextInterface) (string, *ContractError) {
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", wrapError(ErrIdentityUnavailable, err, "MSP-ID der aufrufenden Identität konnte nicht ermittelt werden")
	}
	if mspID == "" {
		return "", newError(ErrIdentityUnavailable, "aufrufende Identität hat eine leere MSP-ID")
	}
	return mspID, nil
}
