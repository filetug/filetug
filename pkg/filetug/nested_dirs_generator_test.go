package filetug

import (
	"context"
	"sync"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGeneratedNestedDirs_DefaultFormat(t *testing.T) {
	var mu sync.Mutex
	paths := []string{}
	store := newMockStore(t)
	store.EXPECT().CreateDir(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, path string) error {
			mu.Lock()
			paths = append(paths, path)
			mu.Unlock()
			return nil
		},
	).AnyTimes()
	err := GeneratedNestedDirs(context.Background(), store, "/root", "", 2, 2)
	assert.NoError(t, err)

	expected := map[string]struct{}{
		"/root":                       {},
		"/root/Directory0":            {},
		"/root/Directory1":            {},
		"/root/Directory0/Directory0": {},
		"/root/Directory0/Directory1": {},
		"/root/Directory1/Directory0": {},
		"/root/Directory1/Directory1": {},
	}

	mu.Lock()
	recordedPaths := append([]string(nil), paths...)
	mu.Unlock()

	assert.Len(t, recordedPaths, len(expected))

	got := make(map[string]struct{}, len(recordedPaths))
	for _, p := range recordedPaths {
		got[p] = struct{}{}
	}
	assert.Len(t, got, len(expected))

	for p := range expected {
		if _, ok := got[p]; !ok {
			t.Errorf("expected path %q to be created", p)
		}
	}
}

func TestGeneratedNestedDirs_DepthZero(t *testing.T) {
	var mu sync.Mutex
	paths := []string{}
	store := newMockStore(t)
	store.EXPECT().CreateDir(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, path string) error {
			mu.Lock()
			paths = append(paths, path)
			mu.Unlock()
			return nil
		},
	).AnyTimes()
	err := GeneratedNestedDirs(context.Background(), store, "/root", "Dir%d", 0, 3)
	assert.NoError(t, err)

	mu.Lock()
	recordedPaths := append([]string(nil), paths...)
	mu.Unlock()

	assert.Equal(t, []string{"/root"}, recordedPaths)
}

func TestNestedDirsGeneratorPanel_GenerateButton(t *testing.T) {
	nav := NewNavigator(nil)

	panel := newNestedDirsGeneratorPanel(nav, nil)
	p, ok := panel.(*nestedDirsGeneratorPanel)
	if !ok {
		t.Fatalf("expected *nestedDirsGeneratorPanel, got %T", panel)
	}

	buttonIndex := p.form.GetButtonIndex("Generate")
	if buttonIndex < 0 {
		t.Fatal("Generate button not found")
	}

	button := p.form.GetButton(buttonIndex)
	handler := button.InputHandler()
	handler(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(_ tview.Primitive) {})
}
