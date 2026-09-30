package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestAppendCtx(t *testing.T) {
	first := slog.String("first", "one")
	second := slog.Int("second", 2)

	ctx := AppendCtx(context.Background(), first)
	ctx = AppendCtx(ctx, second)

	attrs, ok := ctx.Value(slogFields).([]slog.Attr)
	if !ok {
		t.Fatal("expected slog attributes in context")
	}
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	if !attrs[0].Equal(first) || !attrs[1].Equal(second) {
		t.Fatalf("unexpected attributes: %#v", attrs)
	}
}

