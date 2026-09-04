package namespaceregistry

import (
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// maxResolverEndpointLength begrenzt die Grösse von Transaktionsargumenten
// defensiv, um pathologisch grosse Eingaben (versehentlich oder böswillig)
// gar nicht erst in den World State zu übernehmen.
const maxResolverEndpointLength = 2048

// validateNamespace prüft, dass namespace ein syntaktisch korrekter,
// kanonischer UUIDv4 ist (RWP #23: "canonical UUIDv4" als Global Namespace
// Identifier). Grossbuchstaben werden abgelehnt, um genau eine kanonische
// Schreibweise pro Namespace im World State zu garantieren (sonst könnten
// "a3f9e21c-..." und "A3F9E21C-..." als zwei verschiedene Keys existieren,
// obwohl sie denselben Namespace meinen).
func validateNamespace(namespace string) *ContractError {
	if namespace == "" {
		return newError(ErrInvalidNamespace, "namespace darf nicht leer sein")
	}
	if namespace != strings.ToLower(namespace) {
		return newError(ErrInvalidNamespace, "namespace %q muss in Kleinschreibung angegeben werden (kanonische Form)", namespace)
	}
	parsed, err := uuid.Parse(namespace)
	if err != nil {
		return newError(ErrInvalidNamespace, "namespace %q ist kein gültiger UUID: %v", namespace, err)
	}
	if parsed.Version() != 4 {
		return newError(ErrInvalidNamespace, "namespace %q ist kein UUIDv4 (gefundene Version: %d)", namespace, parsed.Version())
	}
	return nil
}

// validateResolverEndpoint prüft, dass resolverEndpoint eine absolute
// HTTPS-URL ist. Reine HTTP-URLs werden abgelehnt: die Registry ist
// öffentlich einsehbar und ein Resolver-Endpoint ohne Transportverschlüsselung
// wäre ein unmittelbares Sicherheitsrisiko für alle, die ihm folgen.
func validateResolverEndpoint(endpoint string) *ContractError {
	if endpoint == "" {
		return newError(ErrInvalidEndpoint, "resolverEndpoint darf nicht leer sein")
	}
	if len(endpoint) > maxResolverEndpointLength {
		return newError(ErrInvalidEndpoint, "resolverEndpoint überschreitet die maximale Länge von %d Zeichen", maxResolverEndpointLength)
	}
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return newError(ErrInvalidEndpoint, "resolverEndpoint %q ist keine gültige URL: %v", endpoint, err)
	}
	if parsed.Scheme != "https" {
		return newError(ErrInvalidEndpoint, "resolverEndpoint %q muss das https-Schema verwenden (gefunden: %q)", endpoint, parsed.Scheme)
	}
	if parsed.Host == "" {
		return newError(ErrInvalidEndpoint, "resolverEndpoint %q enthält keinen Host", endpoint)
	}
	return nil
}
