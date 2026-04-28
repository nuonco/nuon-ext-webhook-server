package payload

import "testing"

func TestCloudEventValidate(t *testing.T) {
	t.Parallel()

	validBase := func() *CloudEvent {
		return &CloudEvent{
			SpecVersion: "1.0",
			ID:          "evt_123",
			Type:        CloudEventType,
			Source:      "//nuon.co/ctl-api",
			Data: &OperationLifecycleEventData{
				Event:     EventPlanStarted,
				Operation: "component-deploy",
				Stage:     StagePlan,
				Status:    StatusStarted,
				Context: OperationContext{
					OrgID: "org_123",
				},
			},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*CloudEvent)
		payload *CloudEvent
		wantErr bool
	}{
		{
			name:    "valid started event",
			mutate:  func(*CloudEvent) {},
			wantErr: false,
		},
		{
			name: "valid finished event with failure",
			mutate: func(ce *CloudEvent) {
				ce.Data.Event = EventApplyFinished
				ce.Data.Stage = StageApply
				ce.Data.Status = StatusFailed
				ce.Data.Error = "boom"
				ce.Data.DurationMs = 1234
			},
			wantErr: false,
		},
		{
			name: "valid single-phase operation",
			mutate: func(ce *CloudEvent) {
				ce.Data.Event = EventOperationStarted
				ce.Data.Stage = ""
				ce.Data.Operation = "install-created"
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			mutate: func(ce *CloudEvent) {
				ce.Type = "com.nuon.signal.lifecycle.v1"
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			mutate: func(ce *CloudEvent) {
				ce.Data.Status = "in_progress"
			},
			wantErr: true,
		},
		{
			name: "invalid event name",
			mutate: func(ce *CloudEvent) {
				ce.Data.Event = "validate.started"
			},
			wantErr: true,
		},
		{
			name: "missing org_id",
			mutate: func(ce *CloudEvent) {
				ce.Data.Context.OrgID = ""
			},
			wantErr: true,
		},
		{
			name: "missing operation",
			mutate: func(ce *CloudEvent) {
				ce.Data.Operation = ""
			},
			wantErr: true,
		},
		{
			name: "missing data",
			mutate: func(ce *CloudEvent) {
				ce.Data = nil
			},
			wantErr: true,
		},
		{
			name: "nil payload",
			mutate: func(ce *CloudEvent) {
				*ce = CloudEvent{}
			},
			payload: nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ce := validBase()
			tc.mutate(ce)

			err := ce.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
		})
	}

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()

		var ce *CloudEvent
		if err := ce.Validate(); err == nil {
			t.Fatalf("expected error for nil payload")
		}
	})
}
