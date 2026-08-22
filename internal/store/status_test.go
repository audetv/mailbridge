package store_test

import (
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestValidStatuses(t *testing.T) {
	statuses := store.ValidStatuses()
	if len(statuses) != 5 {
		t.Errorf("expected 5 statuses, got %d", len(statuses))
	}
}

func TestStatusIsActive(t *testing.T) {
	if !store.StatusNew.IsActive() {
		t.Error("new should be active")
	}
	if !store.StatusInProgress.IsActive() {
		t.Error("in_progress should be active")
	}
	if store.StatusBacklog.IsActive() {
		t.Error("backlog should not be active")
	}
	if store.StatusCompleted.IsActive() {
		t.Error("completed should not be active")
	}
}

func TestStatusIsArchived(t *testing.T) {
	if !store.StatusCompleted.IsArchived() {
		t.Error("completed should be archived")
	}
	if !store.StatusClosed.IsArchived() {
		t.Error("closed should be archived")
	}
	if store.StatusNew.IsArchived() {
		t.Error("new should not be archived")
	}
}
