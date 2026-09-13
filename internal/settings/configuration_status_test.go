package settings

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConfigurationStatusExposesOnlyDatabaseRecovery(t *testing.T) {
	typ := reflect.TypeOf(ConfigurationStatus{})
	for _, name := range []string{"PreservedFields", "PreservedFieldCount"} {
		if _, present := typ.FieldByName(name); present {
			t.Errorf("renderer schema still exposes import-only field %s", name)
		}
	}
	status := ConfigurationStatus{RecoveryRequired: true, ErrorKind: "database_corrupt", Message: "The saved configuration could not be loaded."}
	raw, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"recoveryRequired": true, "errorKind": "database_corrupt", "message": status.Message}
	if !reflect.DeepEqual(wire, want) {
		t.Fatalf("recovery wire contract = %#v", wire)
	}
}
