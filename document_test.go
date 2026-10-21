package sources_test

import (
	"context"
	"errors"
	"testing"

	sources "github.com/aosanya/mwanachama-backend-sources"
)

const handbook = "Intro line.\n\n# Opening hours\n\nThe lab opens at nine.\nIt closes at five.\n\n## Weekends\n\nClosed on Sundays.\n\n# Loans\n\nA laptop goes out for a week.\n"

func TestAMarkdownDocumentSplitsAtHeadingsAndBlankLines(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Lab handbook")

	got, err := mgr.AddDocument(context.Background(), sources.Document{SourceID: src.ID, Text: handbook, Format: "markdown"})
	if err != nil {
		t.Fatalf("AddDocument: %v", err)
	}
	want := []struct{ heading, text string }{
		{"", "Intro line."},
		{"Opening hours", "The lab opens at nine.\nIt closes at five."},
		{"Opening hours / Weekends", "Closed on Sundays."},
		{"Loans", "A laptop goes out for a week."},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d chunks, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		c := got[i]
		if c.Heading != w.heading || c.Text != w.text || c.Position != i {
			t.Fatalf("chunk %d = %+v, want heading %q text %q", i, c, w.heading, w.text)
		}
		if handbook[c.StartOffset:c.EndOffset] != c.Text {
			t.Fatalf("chunk %d offsets %d..%d read %q, not its text", i, c.StartOffset, c.EndOffset, handbook[c.StartOffset:c.EndOffset])
		}
		if len(c.Embedding) != 0 || c.EmbeddingModel != "" {
			t.Fatalf("chunk %d carries an embedding", i)
		}
	}

	stored, err := mgr.ListChunks(context.Background(), src.ID)
	if err != nil {
		t.Fatalf("ListChunks: %v", err)
	}
	if len(stored) != 4 || stored[2].Heading != "Opening hours / Weekends" || stored[2].StartOffset != got[2].StartOffset || stored[2].EndOffset != got[2].EndOffset {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestPlainTextKeepsAHashLineAsText(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Notes")
	got, err := mgr.AddDocument(context.Background(), sources.Document{SourceID: src.ID, Text: "# not a heading\n\nsecond"})
	if err != nil {
		t.Fatalf("AddDocument: %v", err)
	}
	if len(got) != 2 || got[0].Text != "# not a heading" || got[0].Heading != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestADocumentFollowsTheChunksAlreadyHeld(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Notes")
	addChunk(t, mgr, src.ID, 0, "held", nil, "")
	got, err := mgr.AddDocument(context.Background(), sources.Document{SourceID: src.ID, Text: "next"})
	if err != nil {
		t.Fatalf("AddDocument: %v", err)
	}
	if got[0].Position != 1 {
		t.Fatalf("position = %d, want 1", got[0].Position)
	}
}

func TestAFencedHashIsNotAHeading(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Notes")
	got, err := mgr.AddDocument(context.Background(), sources.Document{SourceID: src.ID, Format: "markdown", Text: "# Setup\n\n```\n# comment\n```\n"})
	if err != nil {
		t.Fatalf("AddDocument: %v", err)
	}
	if len(got) != 1 || got[0].Heading != "Setup" || got[0].Text != "```\n# comment\n```" {
		t.Fatalf("got %+v", got)
	}
}

func TestADocumentIsRefused(t *testing.T) {
	mgr := newTestManager(t)
	src := mustSource(t, mgr, "Notes")
	cases := map[string]sources.Document{
		"unknown format": {SourceID: src.ID, Text: "x", Format: "pdf"},
		"empty":          {SourceID: src.ID, Text: "  \n\n"},
		"only headings":  {SourceID: src.ID, Text: "# A\n## B\n", Format: "markdown"},
	}
	for name, in := range cases {
		if _, err := mgr.AddDocument(context.Background(), in); !errors.Is(err, sources.ErrInvalidDocument) {
			t.Fatalf("%s: err = %v, want ErrInvalidDocument", name, err)
		}
	}
	if _, err := mgr.AddDocument(context.Background(), sources.Document{SourceID: "missing", Text: "x"}); !errors.Is(err, sources.ErrSourceNotFound) {
		t.Fatalf("missing source: err = %v", err)
	}
}
