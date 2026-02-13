package filetug

import (
	"context"

	"github.com/filetug/filetug/pkg/files"
)

func (nav *Navigator) delete() {
	b := nav.getCurrentBrowser()
	currentItem := b.GetCurrentEntry()
	if currentItem == nil {
		return
	}
	NewOperation(deleteOperation, func(ctx context.Context, reportProgress ProgressReporter) error {
		currentItemPath := currentItem.FullName()
		if err := deleteEntries(ctx, nav.store, []string{currentItemPath}, reportProgress); err != nil {
			return err
		}
		return nil
	}, nil)
}

const deleteOperation OperationType = "deleteEntries"

func deleteEntries(ctx context.Context, store files.Store, entries []string, _ ProgressReporter) error {
	for _, entry := range entries {
		if err := store.Delete(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}
