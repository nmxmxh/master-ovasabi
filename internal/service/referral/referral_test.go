package service

import (
	"context"
	"testing"
	"time"

	"github.com/ovasabi/master-ovasabi/api/protos/referral"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewReferralService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates service with dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewReferralService(logger)

			assert.NotNil(t, svc)
			assert.NotNil(t, svc.log)
		})
	}
}

func TestReferralService_CreateReferral(t *testing.T) {
	tests := []struct {
		name          string
		request       *referral.CreateReferralRequest
		expectedError error
	}{
		{
			name: "successful referral code creation",
			request: &referral.CreateReferralRequest{
				UserId: "test-user",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewReferralService(zap.NewNop())

			resp, err := svc.CreateReferral(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.ReferralCode)
				assert.Equal(t, "Referral code created successfully", resp.Message)
			}
		})
	}
}

func TestReferralService_ApplyReferral(t *testing.T) {
	tests := []struct {
		name          string
		request       *referral.ApplyReferralRequest
		expectedError error
	}{
		{
			name: "successful referral application",
			request: &referral.ApplyReferralRequest{
				ReferralCode: "MOCK-REF-CODE",
				UserId:       "test-user",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewReferralService(zap.NewNop())

			resp, err := svc.ApplyReferral(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.True(t, resp.Success)
				assert.Equal(t, "Referral applied successfully", resp.Message)
				assert.Equal(t, int32(100), resp.RewardPoints)
			}
		})
	}
}

func TestReferralService_GetReferralStats(t *testing.T) {
	tests := []struct {
		name          string
		request       *referral.GetReferralStatsRequest
		expectedError error
	}{
		{
			name: "successful stats retrieval",
			request: &referral.GetReferralStatsRequest{
				UserId: "test-user",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewReferralService(zap.NewNop())

			resp, err := svc.GetReferralStats(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, int32(5), resp.TotalReferrals)
				assert.Equal(t, int32(3), resp.ActiveReferrals)
				assert.Equal(t, int32(500), resp.TotalRewards)
				require.Len(t, resp.Referrals, 1)

				referral := resp.Referrals[0]
				assert.Equal(t, "MOCK-REF-1", referral.ReferralCode)
				assert.Equal(t, "mock-user-1", referral.ReferredUserId)
				assert.True(t, referral.IsActive)
				assert.Equal(t, int32(100), referral.RewardPoints)
				assert.NotZero(t, referral.CreatedAt)
				assert.True(t, referral.CreatedAt <= time.Now().Unix())
			}
		})
	}
}
