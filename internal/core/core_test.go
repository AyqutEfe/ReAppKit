package core

import (
	"context"
	"errors"
	"testing"
)

func fakeJob(id string, done *bool, calls *int) Job {
	return Job{ID: id, Title: id,
		Check: func(context.Context) (bool, error) { return *done, nil },
		Apply: func(context.Context) error { *calls++; *done = true; return nil },
		Verify: func(context.Context) (bool, error) { return *done, nil },
	}
}

func TestConfirmationAndPreviewNeverApply(t *testing.T) {
	done, calls := false, 0
	jobs := []Job{fakeJob("a", &done, &calls)}
	if _, err := Preview(context.Background(), jobs); err != nil { t.Fatal(err) }
	if result := Execute(context.Background(), jobs, false, nil); result.Error != ErrConfirmationRequired.Error() { t.Fatalf("unexpected result: %+v", result) }
	if calls != 0 { t.Fatalf("applied %d times without confirmation", calls) }
	if result := Execute(context.Background(), jobs, true, nil); result.Items[0].State != Succeeded { t.Fatalf("unexpected result: %+v", result) }
	if result := Execute(context.Background(), jobs, true, nil); result.Items[0].State != Skipped { t.Fatalf("rerun did not skip: %+v", result) }
	if calls != 1 { t.Fatalf("applied %d times", calls) }
}

func TestFailureIsolationAndRetry(t *testing.T) {
	doneA, doneB, doneC, callsA, callsB, callsC := false, false, false, 0, 0, 0
	a := fakeJob("a", &doneA, &callsA)
	a.Apply = func(context.Context) error { callsA++; return errors.New("failure") }
	b := fakeJob("b", &doneB, &callsB)
	b.DependsOn = []string{"a"}
	c := fakeJob("c", &doneC, &callsC)
	jobs := []Job{b, c, a}
	first := Execute(context.Background(), jobs, true, nil)
	if len(first.Items) != 3 || first.Items[0].State != Failed || first.Items[1].State != Blocked || first.Items[2].State != Succeeded { t.Fatalf("unexpected results: %+v", first) }
	if callsB != 0 || callsC != 1 { t.Fatalf("wrong apply calls: b=%d c=%d", callsB, callsC) }
	a.Apply = func(context.Context) error { callsA++; doneA = true; return nil }
	second := RetryFailed(context.Background(), []Job{b, c, a}, first, true, nil)
	if second.Error != "" || second.Items[0].State != Succeeded || second.Items[1].State != Succeeded || second.Items[2].State != Skipped { t.Fatalf("retry result: %+v", second) }
	if callsA != 2 || callsB != 1 || callsC != 1 { t.Fatalf("retry calls: a=%d b=%d c=%d", callsA, callsB, callsC) }
}

func TestInvalidPlanCannotApply(t *testing.T) {
	done, calls := false, 0
	a := fakeJob("a", &done, &calls)
	a.DependsOn = []string{"b"}
	b := fakeJob("b", &done, &calls)
	b.DependsOn = []string{"a"}
	if result := Execute(context.Background(), []Job{a,b}, true, nil); result.Error == "" { t.Fatal("cycle accepted") }
	if calls != 0 { t.Fatal("applied invalid plan") }
}
