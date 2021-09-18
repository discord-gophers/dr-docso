package spec

import (
	"encoding/gob"
	"os"
	"testing"
)

func TestQuerySpec(t *testing.T) {
	spec, err := QuerySpec()
	if err != nil {
		panic(err)
	}

	err = gob.NewEncoder(os.Stdout).Encode(spec)
	if err != nil {
		t.Error(err)
	}
}
