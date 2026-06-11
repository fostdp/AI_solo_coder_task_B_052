package pipeline

import (
	"context"
	"testing"
	"time"
)

type mockStage struct {
	name     string
	received int
	sent     int
}

func (s *mockStage) Name() string { return s.name }

func (s *mockStage) Start(ctx context.Context, in <-chan PipelineMessage, out chan<- PipelineMessage) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-in:
			if !ok {
				return nil
			}
			s.received++
			msg.Metadata.Source = s.name
			select {
			case out <- msg:
				s.sent++
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func TestPipeline_SingleStage(t *testing.T) {
	stage := &mockStage{name: "stage1"}
	p := NewPipeline(stage)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in, out, err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	msg := PipelineMessage{
		Type: MsgTypeRawLoRa,
		Metadata: Metadata{
			TraceID: "test-001",
		},
		Data: "hello",
	}

	select {
	case in <- msg:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout sending message")
	}

	select {
	case result := <-out:
		if result.Metadata.Source != "stage1" {
			t.Errorf("expected source 'stage1', got '%s'", result.Metadata.Source)
		}
		if result.Data != "hello" {
			t.Errorf("expected data 'hello', got '%v'", result.Data)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout receiving message")
	}

	if stage.received != 1 {
		t.Errorf("expected received=1, got %d", stage.received)
	}
}

func TestPipeline_MultiStage(t *testing.T) {
	stage1 := &mockStage{name: "stage1"}
	stage2 := &mockStage{name: "stage2"}
	stage3 := &mockStage{name: "stage3"}

	p := NewPipeline(stage1, stage2, stage3)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in, out, err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	msg := PipelineMessage{
		Type: MsgTypeRawLoRa,
		Metadata: Metadata{
			TraceID: "test-002",
		},
		Data: "payload",
	}

	select {
	case in <- msg:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout sending message")
	}

	select {
	case result := <-out:
		if result.Metadata.Source != "stage3" {
			t.Errorf("expected final source 'stage3', got '%s'", result.Metadata.Source)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout receiving message")
	}

	if stage1.received != 1 || stage2.received != 1 || stage3.received != 1 {
		t.Errorf("each stage should receive 1 msg: stage1=%d stage2=%d stage3=%d",
			stage1.received, stage2.received, stage3.received)
	}
}

func TestPipeline_ContextCancel(t *testing.T) {
	stage := &mockStage{name: "slow"}
	p := NewPipeline(stage)

	ctx, cancel := context.WithCancel(context.Background())

	_, out, err := p.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	cancel()

	done := make(chan struct{})
	go func() {
		for range out {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("pipeline did not stop after context cancel")
	}
}

func TestMessageTypeConstants(t *testing.T) {
	if MsgTypeRawLoRa == MsgTypeDeduplicated {
		t.Error("message type constants should be distinct")
	}
	if MsgTypeDeduplicated == MsgTypeProcessedSensor {
		t.Error("message type constants should be distinct")
	}
	if MsgTypeTermitePrediction == MsgTypeFumigantDiffusion {
		t.Error("message type constants should be distinct")
	}
	if MsgTypeFumigantDiffusion == MsgTypeAlert {
		t.Error("message type constants should be distinct")
	}
}
