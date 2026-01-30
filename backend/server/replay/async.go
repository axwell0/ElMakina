package replay

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"gorm.io/datatypes"
)

type AsyncOptions struct {
	Buffer       int
	Workers      int
	WriteTimeout time.Duration
	DropOnFull   bool
}

type AsyncRecorder struct {
	sink       Recorder
	events     chan EventInput
	snapshots  chan SnapshotInput
	queue      chan func()
	closed     chan struct{}
	opts       AsyncOptions
	batchSize  int
	batchDelay time.Duration
}

const (
	defaultAsyncBuffer       = 256
	defaultAsyncWorkers      = 2
	defaultAsyncWriteTimeout = 2 * time.Second
	defaultBatchSize         = 32
	defaultBatchDelay        = 50 * time.Millisecond
)

func NewAsyncRecorder(sink Recorder, opts AsyncOptions) *AsyncRecorder {
	if opts.Buffer <= 0 {
		opts.Buffer = defaultAsyncBuffer
	}
	if opts.Workers <= 0 {
		opts.Workers = defaultAsyncWorkers
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = defaultAsyncWriteTimeout
	}
	if sink == nil {
		return nil
	}
	r := &AsyncRecorder{
		sink:       sink,
		events:     make(chan EventInput, opts.Buffer),
		snapshots:  make(chan SnapshotInput, opts.Buffer),
		queue:      make(chan func(), opts.Buffer),
		closed:     make(chan struct{}),
		opts:       opts,
		batchSize:  defaultBatchSize,
		batchDelay: defaultBatchDelay,
	}
	for i := 0; i < opts.Workers; i++ {
		go r.worker(opts)
	}
	go r.eventBatcher()
	go r.snapshotBatcher()
	return r
}

func (r *AsyncRecorder) worker(opts AsyncOptions) {
	for {
		select {
		case job := <-r.queue:
			if job != nil {
				job()
			}
		case <-r.closed:
			return
		}
	}
}

func (r *AsyncRecorder) enqueue(job func()) {
	if r == nil {
		return
	}
	if r.opts.DropOnFull {
		select {
		case r.queue <- job:
		default:
			log.Printf("replay async queue full; dropping event")
		}
		return
	}
	select {
	case r.queue <- job:
	case <-time.After(25 * time.Millisecond):
		log.Printf("replay async queue congested; dropping event")
	}
}

func (r *AsyncRecorder) eventBatcher() {
	ticker := time.NewTicker(r.batchDelay)
	defer ticker.Stop()
	buffer := make([]EventInput, 0, r.batchSize)
	for {
		select {
		case event := <-r.events:
			buffer = append(buffer, event)
			if len(buffer) >= r.batchSize {
				r.flushEvents(buffer)
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				r.flushEvents(buffer)
				buffer = buffer[:0]
			}
		case <-r.closed:
			if len(buffer) > 0 {
				r.flushEvents(buffer)
			}
			return
		}
	}
}

func (r *AsyncRecorder) snapshotBatcher() {
	ticker := time.NewTicker(r.batchDelay)
	defer ticker.Stop()
	buffer := make([]SnapshotInput, 0, r.batchSize)
	for {
		select {
		case snapshot := <-r.snapshots:
			buffer = append(buffer, snapshot)
			if len(buffer) >= r.batchSize {
				r.flushSnapshots(buffer)
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				r.flushSnapshots(buffer)
				buffer = buffer[:0]
			}
		case <-r.closed:
			if len(buffer) > 0 {
				r.flushSnapshots(buffer)
			}
			return
		}
	}
}

func (r *AsyncRecorder) flushEvents(inputs []EventInput) {
	if r == nil || len(inputs) == 0 {
		return
	}
	store, ok := r.sink.(*Store)
	if !ok {
		for _, input := range inputs {
			_ = r.sink.RecordEvent(context.Background(), input)
		}
		return
	}
	events := make([]MatchEvent, 0, len(inputs))
	for _, input := range inputs {
		payload, err := json.Marshal(input.Payload)
		if err != nil {
			log.Printf("replay async event marshal failed: %v", err)
			continue
		}
		events = append(events, MatchEvent{
			MatchID:         input.MatchID,
			Seq:             input.Seq,
			Type:            input.Type,
			Visibility:      input.Visibility,
			PlayerID:        input.PlayerID,
			Payload:         datatypes.JSON(payload),
			RulesetVersion:  input.RulesetVersion,
			ProtocolVersion: input.ProtocolVersion,
			CreatedAt:       time.Now(),
		})
	}
	if err := store.recordEventsBatch(context.Background(), events); err != nil {
		log.Printf("replay async batch insert failed: %v", err)
	}
}

func (r *AsyncRecorder) flushSnapshots(inputs []SnapshotInput) {
	if r == nil || len(inputs) == 0 {
		return
	}
	store, ok := r.sink.(*Store)
	if !ok {
		for _, input := range inputs {
			_ = r.sink.Snapshot(context.Background(), input)
		}
		return
	}
	snapshots := make([]MatchSnapshot, 0, len(inputs))
	for _, input := range inputs {
		if input.State == nil {
			continue
		}
		payload, err := json.Marshal(input.State)
		if err != nil {
			log.Printf("replay async snapshot marshal failed: %v", err)
			continue
		}
		snapshots = append(snapshots, MatchSnapshot{
			MatchID:   input.MatchID,
			Seq:       input.Seq,
			Payload:   datatypes.JSON(payload),
			CreatedAt: time.Now(),
		})
	}
	if err := store.recordSnapshotsBatch(context.Background(), snapshots); err != nil {
		log.Printf("replay async snapshot batch insert failed: %v", err)
	}
}

func (r *AsyncRecorder) withTimeout(op func(ctx context.Context) error, timeout time.Duration) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := op(ctx); err != nil {
			log.Printf("replay async write failed: %v", err)
		}
	}
}

func (r *AsyncRecorder) StartMatch(ctx context.Context, input MatchStartInput) error {
	if r == nil {
		return nil
	}
	r.enqueue(r.withTimeout(func(ctx context.Context) error {
		return r.sink.StartMatch(ctx, input)
	}, r.opts.WriteTimeout))
	return nil
}

func (r *AsyncRecorder) RecordEvent(ctx context.Context, input EventInput) error {
	if r == nil {
		return nil
	}
	if r.opts.DropOnFull {
		select {
		case r.events <- input:
		default:
			log.Printf("replay async event queue full; dropping event")
		}
		return nil
	}
	select {
	case r.events <- input:
	case <-time.After(25 * time.Millisecond):
		log.Printf("replay async event queue congested; dropping event")
	}
	return nil
}

func (r *AsyncRecorder) Snapshot(ctx context.Context, input SnapshotInput) error {
	if r == nil {
		return nil
	}
	if r.opts.DropOnFull {
		select {
		case r.snapshots <- input:
		default:
			log.Printf("replay async snapshot queue full; dropping snapshot")
		}
		return nil
	}
	select {
	case r.snapshots <- input:
	case <-time.After(25 * time.Millisecond):
		log.Printf("replay async snapshot queue congested; dropping snapshot")
	}
	return nil
}

func (r *AsyncRecorder) EndMatch(ctx context.Context, input MatchEndInput) error {
	if r == nil {
		return nil
	}
	r.enqueue(r.withTimeout(func(ctx context.Context) error {
		return r.sink.EndMatch(ctx, input)
	}, r.opts.WriteTimeout))
	return nil
}

func (r *AsyncRecorder) LoadReplay(ctx context.Context, matchID string, viewerPlayerID string) (*ReplayPayload, error) {
	if r == nil {
		return nil, nil
	}
	return r.sink.LoadReplay(ctx, matchID, viewerPlayerID)
}

// Close stops async workers (best-effort). Not part of the Recorder interface.
func (r *AsyncRecorder) Close() {
	if r == nil {
		return
	}
	close(r.closed)
}
