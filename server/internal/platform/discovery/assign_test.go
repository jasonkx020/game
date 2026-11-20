package discovery

import "testing"

func TestAssignStickyAndExclude(t *testing.T) {
	live := []Connector{
		{ID: "a", WSURL: "ws://a"},
		{ID: "b", WSURL: "ws://b"},
		{ID: "c", WSURL: "ws://c"},
	}
	first := Assign(42, live, "")
	if first == nil {
		t.Fatal("expected connector")
	}
	again := Assign(42, live, "")
	if again == nil || again.ID != first.ID {
		t.Fatalf("sticky: got %v want %s", again, first.ID)
	}
	other := Assign(42, live, first.ID)
	if other == nil {
		t.Fatal("expected alternate after exclude")
	}
	if other.ID == first.ID {
		t.Fatalf("exclude failed: still %s", other.ID)
	}
}

func TestAssignEmpty(t *testing.T) {
	if Assign(1, nil, "") != nil {
		t.Fatal("expected nil")
	}
	if Assign(1, []Connector{{ID: "only", WSURL: "ws://x"}}, "only") != nil {
		t.Fatal("expected nil when all excluded")
	}
}
