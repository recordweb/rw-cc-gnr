package namespaceregistry

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateNamespace_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		namespace string
		wantErr   bool
		wantCode  ErrorCode
	}{
		{"gültiger UUIDv4 lowercase", "a3f9e21c-1234-4abc-8def-1234567890ab", false, ""},
		{"leerer String", "", true, ErrInvalidNamespace},
		{"kein UUID", "nicht-ein-uuid", true, ErrInvalidNamespace},
		{"UUID Grossbuchstaben", "A3F9E21C-1234-4ABC-8DEF-1234567890AB", true, ErrInvalidNamespace},
		{"UUIDv1 statt v4", "a3f9e21c-1234-1abc-8def-1234567890ab", true, ErrInvalidNamespace},
		{"UUID mit zusätzlichen Zeichen", "a3f9e21c-1234-4abc-8def-1234567890ab-extra", true, ErrInvalidNamespace},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNamespace(tc.namespace)
			if !tc.wantErr {
				require.Nil(t, err)
				return
			}
			require.NotNil(t, err)
			require.Equal(t, tc.wantCode, err.Code)
		})
	}
}

func TestValidateResolverEndpoint_TableDriven(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{"gültige https URL", "https://resolver.example.org/rwp/1.0/identifiers", false},
		{"leerer String", "", true},
		{"http statt https", "http://insecure.example.org", true},
		{"kein Host", "https://", true},
		{"ungültige URL", "not a url at all", true},
		{"zu lang", "https://example.org/" + strings.Repeat("a", 3000), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateResolverEndpoint(tc.endpoint)
			if tc.wantErr {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
