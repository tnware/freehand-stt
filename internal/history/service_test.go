package history

import (
	"testing"
	"time"
)

func TestDetailsSelectionFollowsRetainedHistory(t *testing.T) {
	store := NewStore(true, nil)
	opened, hidden, changed := 0, 0, 0
	service := NewService(store, DetailsWindow{
		Open: func() { opened++ }, Hide: func() { hidden++ }, Changed: func() { changed++ },
	})
	add := func(text string) uint64 {
		return store.Begin(text, HistoryTranscribed, false, time.Now(), HistoryRunDetails{
			CompletedAt: time.Now(), Segments: []HistorySegmentDetails{{Boundary: "silence"}},
		})
	}
	first, second := add("first"), add("second")
	if err := service.OpenDetails(first); err != nil {
		t.Fatal(err)
	}
	entry := service.CurrentDetails()
	if entry == nil || entry.ID != first {
		t.Fatal("first selection missing")
	}
	entry.Details.Segments[0].Boundary = "changed"
	if service.CurrentDetails().Details.Segments[0].Boundary != "silence" {
		t.Fatal("details mutated retained history")
	}
	if err := service.OpenDetails(second); err != nil {
		t.Fatal(err)
	}
	if service.CurrentDetails().ID != second || opened != 2 {
		t.Fatal("selection did not switch")
	}
	if err := service.OpenDetails(0); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if service.CurrentDetails().ID != second {
		t.Fatal("invalid request changed selection")
	}
	if err := service.DeleteHistoryEntry(second); err != nil {
		t.Fatal(err)
	}
	if service.CurrentDetails() != nil {
		t.Fatal("deleted entry retained by details")
	}
	if err := service.OpenDetails(first); err != nil {
		t.Fatal(err)
	}
	service.CloseDetails()
	if hidden != 1 || service.CurrentDetails() != nil {
		t.Fatal("close retained selection")
	}
	if err := service.OpenDetails(first); err != nil {
		t.Fatal(err)
	}
	service.ClearHistory()
	if service.CurrentDetails() != nil || changed != 7 {
		t.Fatal("clear did not invalidate details")
	}
}

func TestDetailsRejectUnfinishedAndRemovedHistory(t *testing.T) {
	for _, removal := range []string{"disable", "evict", "shutdown"} {
		t.Run(removal, func(t *testing.T) {
			store := NewStore(true, nil)
			service := NewService(store, DetailsWindow{Open: func() {}})
			pending := store.Begin("pending", HistoryTranscribed, true, time.Now(), HistoryRunDetails{CompletedAt: time.Now()})
			if err := service.OpenDetails(pending); err == nil {
				t.Fatal("pending cleanup accepted")
			}
			unfinished := store.Begin("unfinished", HistoryTranscribed, false, time.Now(), HistoryRunDetails{})
			if err := service.OpenDetails(unfinished); err == nil {
				t.Fatal("unfinished run accepted")
			}
			id := store.Begin("done", HistoryTranscribed, false, time.Now(), HistoryRunDetails{CompletedAt: time.Now()})
			if err := service.OpenDetails(id); err != nil {
				t.Fatal(err)
			}
			switch removal {
			case "disable":
				store.SetEnabled(false)
			case "evict":
				for i := 0; i < MaxHistoryEntries; i++ {
					store.Begin("later", HistoryTranscribed, false, time.Now(), HistoryRunDetails{CompletedAt: time.Now()})
				}
			case "shutdown":
				_ = service.ServiceShutdown()
			}
			if service.CurrentDetails() != nil {
				t.Fatal("removed history retained by details")
			}
			if err := service.OpenDetails(id); err == nil {
				t.Fatal("removed history opened")
			}
		})
	}
}
