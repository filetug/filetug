package filetug

import (
	"context"
)

type OperationType string

type OperationProgress struct {
	Total      int
	Done       int
	Failed     int
	Skipped    int
	Processing []string
}

type Operation struct {
	Type     OperationType
	cancel   context.CancelFunc
	done     chan error
	progress OperationProgress
}

type ProgressReporter = func(progress OperationProgress)

func NewOperation(
	t OperationType,
	f func(ctx context.Context, reportProgress ProgressReporter) error,
	reportProgress ProgressReporter,
) *Operation {
	o := &Operation{Type: t}
	ctx := context.Background()
	ctx, o.cancel = context.WithCancel(ctx)
	go func() {
		reportOperProgress := func(progress OperationProgress) {
			o.progress = progress
			if reportProgress != nil {
				reportProgress(progress)
			}
		}
		err := f(ctx, reportOperProgress)
		o.done <- err
	}()
	return o
}
