package ws

import (
	"testing"

	"ElMakina/backend/models"
)

func TestTargetIndexFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *models.PlayerAction
		want *int
	}{
		{
			name: "nil command",
			cmd:  nil,
			want: nil,
		},
		{
			name: "target payload",
			cmd: &models.PlayerAction{
				ID:      models.Steal,
				Payload: models.TargetPayload{TargetIndex: 2},
			},
			want: intPtr(2),
		},
		{
			name: "accuse payload",
			cmd: &models.PlayerAction{
				ID:      models.Accuse,
				Payload: models.AccusePayload{TargetIndex: 1, Guess: models.Thief},
			},
			want: intPtr(1),
		},
		{
			name: "unknown payload",
			cmd: &models.PlayerAction{
				ID:      models.Income,
				Payload: "noop",
			},
			want: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := targetIndexFromAction(tc.cmd)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
			if got != nil && *got != *tc.want {
				t.Fatalf("expected %d, got %d", *tc.want, *got)
			}
		})
	}
}
