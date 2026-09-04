package namespaceregistry

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// logEntry ist die strukturierte Log-Zeile, die nach stdout geschrieben wird.
// Peers leiten Chaincode-Container-stdout in ihre eigenen Logs weiter;
// ein JSON-Format erlaubt späteres Einsammeln durch z.B. Loki/ELK, ohne
// Freitext-Logs parsen zu müssen.
type logEntry struct {
	Level     string                 `json:"level"`
	Function  string                 `json:"function"`
	TxID      string                 `json:"txId,omitempty"`
	MSPID     string                 `json:"mspId,omitempty"`
	Namespace string                 `json:"namespace,omitempty"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// chaincodeLogger schreibt strukturierte JSON-Logs nach stdout/stderr.
// Bewusst kein externes Logging-Framework: Chaincode-Container haben
// keine garantierte Netzwerkverbindung nach aussen, daher ist stdout/stderr
// (vom Peer eingesammelt) der einzige verlässliche Log-Kanal.
type chaincodeLogger struct {
	std *log.Logger
	err *log.Logger
}

var logger = newChaincodeLogger()

func newChaincodeLogger() *chaincodeLogger {
	return &chaincodeLogger{
		std: log.New(os.Stdout, "", 0),
		err: log.New(os.Stderr, "", 0),
	}
}

func (l *chaincodeLogger) write(w *log.Logger, level, function, txID, mspID, namespace, message string, fields map[string]interface{}) {
	entry := logEntry{
		Level:     level,
		Function:  function,
		TxID:      txID,
		MSPID:     mspID,
		Namespace: namespace,
		Message:   message,
		Fields:    fields,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback: niemals einen Logging-Fehler die eigentliche
		// Transaktion beeinträchtigen lassen.
		w.Printf(`{"level":"%s","function":"%s","message":"log marshal failed: %v"}`, level, function, err)
		return
	}
	w.Println(string(data))
}

// Info loggt einen normalen Ablaufschritt (z.B. erfolgreiche Registrierung).
func (l *chaincodeLogger) Info(function, txID, mspID, namespace, message string, fields map[string]interface{}) {
	l.write(l.std, "INFO", function, txID, mspID, namespace, message, fields)
}

// Warn loggt einen erwarteten, aber bemerkenswerten Zustand (z.B. Duplikat-Versuch).
func (l *chaincodeLogger) Warn(function, txID, mspID, namespace, message string, fields map[string]interface{}) {
	l.write(l.std, "WARN", function, txID, mspID, namespace, message, fields)
}

// Error loggt einen tatsächlichen Fehlerzustand (z.B. Ledger-I/O-Fehler).
func (l *chaincodeLogger) Error(function, txID, mspID, namespace, message string, fields map[string]interface{}) {
	l.write(l.err, "ERROR", function, txID, mspID, namespace, message, fields)
}

// fieldsWithError ist ein kleiner Helfer, um einen error konsistent als
// String-Feld in die strukturierten Log-Felder einzubetten.
func fieldsWithError(err error) map[string]interface{} {
	if err == nil {
		return nil
	}
	return map[string]interface{}{"error": fmt.Sprintf("%v", err)}
}
