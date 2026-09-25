package capturekit

import (
	"context"
	"errors"
	"testing"

	"github.com/amp-labs/connectors"
	"github.com/amp-labs/connectors/common"
)

type fakeWriter struct {
	connectors.Connector

	result *common.WriteResult
	err    error
	got    common.WriteParams
}

func (f *fakeWriter) Write(_ context.Context, p common.WriteParams) (*common.WriteResult, error) {
	f.got = p
	return f.result, f.err
}

func TestWriteOneKeepsTheCreatedRecordID(t *testing.T) {
	t.Parallel()

	w := &fakeWriter{result: &common.WriteResult{Success: true, RecordId: "rec-123"}}
	rec := writeOne(context.Background(), w, "contacts", map[string]any{"name": "x"}, newCapturingClient())

	if rec.Object != "write:contacts" || rec.Result.Status != "ok" || rec.Result.RecordId != "rec-123" {
		t.Fatalf("unexpected capture: %+v", rec)
	}
	if w.got.RecordId != "" || w.got.ObjectName != "contacts" {
		t.Fatalf("must be a create of the named object, got %+v", w.got)
	}
}

func TestWriteOneRecordsFailuresVerbatim(t *testing.T) {
	t.Parallel()

	rec := writeOne(context.Background(), &fakeWriter{err: errors.New("400: name required")},
		"contacts", nil, newCapturingClient())
	if rec.Result.Status != "error" || rec.Result.Error != "400: name required" {
		t.Fatalf("unexpected capture: %+v", rec.Result)
	}

	rec = writeOne(context.Background(), &fakeWriter{result: &common.WriteResult{Success: false}},
		"contacts", nil, newCapturingClient())
	if rec.Result.Status != "error" {
		t.Fatalf("Success=false must be an error, got %+v", rec.Result)
	}
}
