// Package core executes selected setup jobs in dependency order.
package core

import (
	"context"
	"errors"
	"fmt"
)

type State string

const (
	Pending State = "pending"
	Skipped State = "skipped"
	Succeeded State = "succeeded"
	Failed State = "failed"
	Blocked State = "blocked"
)

// Check and Verify must be read-only. Apply is invoked only after confirmation.
type Job struct {
	ID, Title string
	DependsOn []string
	Check func(context.Context) (bool, error)
	Apply func(context.Context) error
	Verify func(context.Context) (bool, error)
}

type Item struct {
	ID string `json:"id"`
	Title string `json:"title"`
	State State `json:"state"`
	Error string `json:"error,omitempty"`
}

type Result struct {
	Items []Item `json:"items"`
	Error string `json:"error,omitempty"`
}

var ErrConfirmationRequired = errors.New("explicit confirmation required")

// Validate rejects malformed jobs and cycles before any checks or changes.
// Unrelated jobs retain their input order.
func Validate(jobs []Job) ([]Job, error) {
	byID := make(map[string]Job, len(jobs))
	for _, j := range jobs {
		if j.ID == "" || j.Title == "" || j.Check == nil || j.Apply == nil || j.Verify == nil {
			return nil, fmt.Errorf("incomplete job %q", j.ID)
		}
		if _, exists := byID[j.ID]; exists { return nil, fmt.Errorf("duplicate job %q", j.ID) }
		byID[j.ID] = j
	}
	marks := make(map[string]uint8, len(jobs))
	ordered := make([]Job, 0, len(jobs))
	var visit func(string) error
	visit = func(id string) error {
		if marks[id] == 2 { return nil }
		if marks[id] == 1 { return fmt.Errorf("dependency cycle at %q", id) }
		j, exists := byID[id]
		if !exists { return fmt.Errorf("unknown dependency %q", id) }
		marks[id] = 1
		for _, dep := range j.DependsOn { if err := visit(dep); err != nil { return err } }
		marks[id] = 2
		ordered = append(ordered, j)
		return nil
	}
	for _, j := range jobs { if err := visit(j.ID); err != nil { return nil, err } }
	return ordered, nil
}

// Preview calls only Check and never invokes Apply or Verify.
func Preview(ctx context.Context, jobs []Job) ([]Item, error) {
	ordered, err := Validate(jobs)
	if err != nil { return nil, err }
	items := make([]Item, 0, len(ordered))
	for _, j := range ordered {
		item := Item{ID: j.ID, Title: j.Title, State: Pending}
		if err := ctx.Err(); err != nil { item.State, item.Error = Failed, err.Error() } else if done, err := j.Check(ctx); err != nil {
			item.State, item.Error = Failed, err.Error()
		} else if done { item.State = Skipped }
		items = append(items, item)
	}
	return items, nil
}

// Execute checks, applies and verifies in dependency order. A failed job blocks
// dependents but independent jobs continue. False confirmation prevents changes.
func Execute(ctx context.Context, jobs []Job, confirmed bool, onEvent func(Item)) Result {
	if !confirmed { return Result{Error: ErrConfirmationRequired.Error()} }
	ordered, err := Validate(jobs)
	if err != nil { return Result{Error: err.Error()} }
	result := Result{Items: make([]Item, 0, len(ordered))}
	states := make(map[string]State, len(ordered))
	emit := func(item Item) {
		states[item.ID] = item.State
		result.Items = append(result.Items, item)
		if onEvent != nil { onEvent(item) }
	}
	for _, j := range ordered {
		item := Item{ID: j.ID, Title: j.Title, State: Pending}
		for _, dep := range j.DependsOn {
			if states[dep] != Succeeded && states[dep] != Skipped {
				item.State, item.Error = Blocked, fmt.Sprintf("dependency %s did not complete", dep)
				break
			}
		}
		if item.State == Blocked { emit(item); continue }
		if err := ctx.Err(); err != nil { item.State, item.Error = Failed, err.Error(); emit(item); continue }
		if done, err := j.Check(ctx); err != nil {
			item.State, item.Error = Failed, fmt.Sprintf("check: %v", err)
		} else if done {
			item.State = Skipped
		} else if err := j.Apply(ctx); err != nil {
			item.State, item.Error = Failed, fmt.Sprintf("apply: %v", err)
		} else if verified, err := j.Verify(ctx); err != nil {
			item.State, item.Error = Failed, fmt.Sprintf("verify: %v", err)
		} else if !verified {
			item.State, item.Error = Failed, "verify: target state not reached"
		} else {
			item.State = Succeeded
		}
		emit(item)
	}
	return result
}

// RetryFailed rechecks completed work and retries failed or blocked work.
func RetryFailed(ctx context.Context, jobs []Job, previous Result, confirmed bool, onEvent func(Item)) Result {
	if !confirmed { return Result{Error: ErrConfirmationRequired.Error()} }
	ordered, err := Validate(jobs)
	if err != nil { return Result{Error: err.Error()} }
	prior := make(map[string]State, len(previous.Items))
	for _, item := range previous.Items { prior[item.ID] = item.State }
	for _, j := range ordered { if _, ok := prior[j.ID]; !ok { return Result{Error: fmt.Sprintf("previous result missing %q", j.ID)} } }
	return Execute(ctx, ordered, true, onEvent)
}
