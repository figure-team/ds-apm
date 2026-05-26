package mockrunbookdrafter

import (
	"context"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/ruletypes"
)

func TestMockDrafter_ReturnsFixedRunbook(t *testing.T) {
	want := ruletypes.Runbook{
		ID:    "01928374-5566-77ab-89cd-eeff00112233",
		Title: "Mock restart",
	}
	d := New(want)
	got, err := d.Draft(context.Background(), ruletypes.RunbookDraftRequest{})
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if got.ID != want.ID || got.Title != want.Title {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestMockDrafter_ReturnsErrorWhenSet(t *testing.T) {
	d := NewError("simulated auth failure")
	_, err := d.Draft(context.Background(), ruletypes.RunbookDraftRequest{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
