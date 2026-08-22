package adapters_test

import (
	"testing"

	"github.com/audetv/mailbridge/internal/adapters"
)

func TestAdapterInterface(t *testing.T) {
	var adapter adapters.Adapter
	if adapter != nil {
		t.Fatal("adapter should be nil")
	}
}
