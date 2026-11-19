package agentdb

import (
	"path/filepath"
	"testing"
)

func TestTodoDedupeKeepsStatus(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	res := db.UpsertTodos([]Todo{{Title: "回复客户", Kind: "todo", CID: "c1"}, {Title: "跟进测试", Kind: "follow-up"}})
	if !res[0].Created || !res[1].Created || res[1].Todo.Kind != "follow_up" {
		t.Fatalf("unexpected create results: %+v", res)
	}
	done := "done"
	if _, err := db.UpdateTodo(res[0].Todo.ID, TodoPatch{Status: &done}); err != nil {
		t.Fatal(err)
	}
	// Same item again (e.g. next day's run) must not reopen or duplicate it.
	again := db.UpsertTodos([]Todo{{Title: "回复客户", Kind: "todo", CID: "c1", Detail: "新细节"}})
	if again[0].Created || again[0].Todo.Status != "done" || again[0].Todo.Detail != "新细节" {
		t.Fatalf("dedupe failed: %+v", again[0])
	}
	open, _ := db.ListTodos(TodoFilter{})
	all, _ := db.ListTodos(TodoFilter{Status: "all"})
	if len(open) != 1 || len(all) != 2 {
		t.Fatalf("open=%d all=%d", len(open), len(all))
	}
	if r := db.UpsertTodos([]Todo{{Title: " "}}); r[0].Error == "" {
		t.Fatal("expected validation error for empty title")
	}
}

func TestSummaryUpsert(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.SaveSummary(Summary{DateKey: "2026-11-19", Summary: "v1"}); err != nil {
		t.Fatal(err)
	}
	s, err := db.SaveSummary(Summary{DateKey: "2026-11-19", Summary: "v2", Focus: []string{"a"}})
	if err != nil || s.Summary != "v2" || len(s.Focus) != 1 {
		t.Fatalf("got %+v, %v", s, err)
	}
	if s, _ := db.GetSummary("2026-11-18"); s != nil {
		t.Fatal("expected nil for missing date")
	}
}
