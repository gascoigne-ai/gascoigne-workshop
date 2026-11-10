package serial

import "testing"

func TestASecondWriteWaitsForTheRunningOneAndTheNewestWins(t *testing.T) {
	var q Queue
	var ran []string
	job := func(label string) Job { return Job{Run: func() error { ran = append(ran, label); return nil }} }
	if !q.Submit("collection", job("a")) {
		t.Fatal("the first write starts at once")
	}
	if q.Submit("collection", job("b")) || q.Submit("collection", job("c")) {
		t.Fatal("a write asked for during a write waits")
	}
	if !q.Submit("settings", job("s")) {
		t.Fatal("another file is not held up")
	}
	next, ok := q.Finish("collection")
	if !ok {
		t.Fatal("the waiting write comes next")
	}
	next.Run()
	if len(ran) != 1 || ran[0] != "c" {
		t.Fatalf("ran %v, want only the newest waiting write", ran)
	}
	if _, ok := q.Finish("collection"); ok {
		t.Fatal("nothing waits after it")
	}
	if !q.Busy() {
		t.Fatal("settings is still running")
	}
	q.Finish("settings")
	if q.Busy() {
		t.Fatal("all finished")
	}
}
