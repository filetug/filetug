package ftui

import "testing"

func TestMenuItem(t *testing.T) {
	t.Parallel()
	called := false
	action := func() {
		called = true
	}
	item := MenuItem{
		Title:   "Test",
		HotKeys: []string{"Ctrl-T"},
		Action:  action,
	}

	if item.Title != "Test" {
		t.Errorf("expected Title Test, got %s", item.Title)
	}
	if len(item.HotKeys) != 1 || item.HotKeys[0] != "Ctrl-T" {
		t.Errorf("expected HotKeys [Ctrl-T], got %v", item.HotKeys)
	}
	item.Action()
	if !called {
		t.Error("expected Action to be called")
	}
}
