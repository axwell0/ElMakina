package tests

import (
	"ElMakina/backend/actions"
	"ElMakina/backend/engine/state"
	"ElMakina/backend/models"
	"context"
	"testing"
)

func TestActionFlow(t *testing.T) {
	// A mock action that yields twice
	mockAction := func(f *actions.ActionFlow) error {
		// First Yield - using helper
		indices, err := f.RequestCardSelection(models.Income, 0, 1, state.ContextExchange, []string{"Card 1"})
		if err != nil {
			return err
		}
		if indices[0] != 42 {
			t.Errorf("expected 42, got %v", indices[0])
		}

		// Second Yield - manual yield for a custom kind
		resp2 := f.Yield(state.StepRequest{
			ActionID: models.Income,
			Kind:     "confirm",
		})
		val2 := resp2.(string)
		if val2 != "ok" {
			t.Errorf("expected 'ok', got %v", val2)
		}

		return nil
	}

	runner := actions.StartAction(context.Background(), mockAction)

	// Get first request
	req1 := <-runner.Flow.Requests
	if req1.Count != 1 {
		t.Fatalf("expected count 1, got %d", req1.Count)
	}

	// Submit first result and get second request
	req2, err := runner.Next([]int{42})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req2.Kind != "confirm" {
		t.Fatalf("expected confirm, got %s", req2.Kind)
	}

	// Submit final result and expect completion
	finalReq, err := runner.Next("ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finalReq != nil {
		t.Fatalf("expected nil request (completion), got %+v", finalReq)
	}
}

func TestActionFlowPersistenceReplay(t *testing.T) {
	// A complex action with side effects and multiple yields
	counter := 0
	complexAction := func(f *actions.ActionFlow) error {
		counter++ // Side effect 1
		f.Yield(state.StepRequest{Kind: state.StepKind("step1")})

		counter++ // Side effect 2
		f.Yield(state.StepRequest{Kind: state.StepKind("step2")})

		counter++ // Side effect 3
		return nil
	}

	// 1. First run - stopped after step 1
	r1 := actions.StartAction(context.Background(), complexAction)
	<-r1.Flow.Requests            // Get step1 (initial yield)
	req2, err := r1.Next("data1") // Resume and get step2 yield
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req2.Kind != "step2" {
		t.Fatalf("expected step2, got %s", req2.Kind)
	}
	// Simulation: CRASH HERE. r1 is lost. counter is 2.

	// 2. Replay run - starting fresh but feeding same data
	counter = 0 // Reset side effects to simulate fresh restart
	r2 := actions.StartAction(context.Background(), complexAction)

	// Fast-forward through step 1 using recorded data
	<-r2.Flow.Requests
	reqReplay2, _ := r2.Next("data1")

	if reqReplay2.Kind != "step2" {
		t.Fatal("Replay failed to reach step2")
	}
	if counter != 2 {
		t.Fatalf("Side effects not re-applied correctly: %d", counter)
	}

	// Finalize to check end state
	r2.Next("data2")
	if counter != 3 {
		t.Fatalf("Side effects not finished correctly: %d", counter)
	}
}
