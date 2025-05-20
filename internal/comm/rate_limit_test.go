package comm

import (
	"reflect"
	"testing"
	"time"
)

func TestRateLimitRequestData(t *testing.T) {
	type fields struct {
		Rate   uint64
		Burst  uint64
		Period time.Duration
		Key    string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "TestRateLimitRequestData",
			fields: fields{
				Rate:   100,
				Burst:  100,
				Period: time.Hour,
				Key:    "testing",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RateLimitRequestData{
				Rate:   tt.fields.Rate,
				Burst:  tt.fields.Burst,
				Period: tt.fields.Period,
				Key:    tt.fields.Key,
			}
			marshalled := r.Marshall()
			unmarshalled := &RateLimitRequestData{}
			err := unmarshalled.Unmarshal(marshalled)
			if err != nil {
				t.Errorf("failed to unmarshal: %v", err)
				return
			}
			if !reflect.DeepEqual(unmarshalled, r) {
				t.Errorf("Expected %v \nWanted %v", unmarshalled, r)
			}
		})
	}
}

func TestRateLimitResponseData(t *testing.T) {
	type fields struct {
		Allowed    uint64
		Remaining  uint64
		RetryAfter time.Duration
		ResetAfter time.Duration
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "TestRateLimitResponseData",
			fields: fields{
				Allowed:    100,
				Remaining:  100,
				RetryAfter: time.Hour,
				ResetAfter: time.Hour,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RateLimitResponseData{
				Allowed:    tt.fields.Allowed,
				Remaining:  tt.fields.Remaining,
				RetryAfter: tt.fields.RetryAfter,
				ResetAfter: tt.fields.ResetAfter,
			}
			marshalled := r.Marshall()
			unmarshalled := &RateLimitResponseData{}
			err := unmarshalled.Unmarshal(marshalled)
			if err != nil {
				t.Errorf("failed to unmarshal: %v", err)
				return
			}
			if !reflect.DeepEqual(unmarshalled, r) {
				t.Errorf("Expected %v \nWanted %v", unmarshalled, r)
			}
		})
	}
}
