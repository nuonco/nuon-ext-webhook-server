package payload

import "testing"

func TestStepLifecycleWebhookPayloadValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload *StepLifecycleWebhookPayload
		wantErr bool
	}{
		{
			name: "valid before payload",
			payload: &StepLifecycleWebhookPayload{
				Version:    "v1",
				Transition: TransitionBefore,
				Event: StepPhaseEvent{
					WorkflowID: "wf_123",
					StepID:     "step_123",
					Phase:      "execute",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid transition",
			payload: &StepLifecycleWebhookPayload{
				Version:    "v1",
				Transition: "during",
				Event: StepPhaseEvent{
					WorkflowID: "wf_123",
					StepID:     "step_123",
					Phase:      "execute",
				},
			},
			wantErr: true,
		},
		{
			name: "missing workflow id",
			payload: &StepLifecycleWebhookPayload{
				Version:    "v1",
				Transition: TransitionAfter,
				Event: StepPhaseEvent{
					StepID: "step_123",
					Phase:  "finalize",
				},
			},
			wantErr: true,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.payload.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
		})
	}
}
