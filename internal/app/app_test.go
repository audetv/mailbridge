// Package app_test содержит тесты для пакета app.
package app_test

import (
	"testing"
)

func TestPlaceholder(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math is broken")
	}
}
