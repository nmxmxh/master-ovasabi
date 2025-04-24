package user

import (
	"context"
	"testing"

	userpb "github.com/ovasabi/master-ovasabi/api/protos/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewUserService(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)
	assert.NotNil(t, service, "Service should not be nil")
}

func TestCreateUser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	tests := []struct {
		name    string
		req     *userpb.CreateUserRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful user creation",
			req: &userpb.CreateUserRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Roles:    []string{"user"},
				Profile: &userpb.UserProfile{
					FirstName: "Test",
					LastName:  "User",
				},
				Metadata: map[string]string{
					"source": "test",
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			req: &userpb.CreateUserRequest{
				Email:    "test@example.com",
				Username: "testuser2",
			},
			wantErr: true,
			errCode: codes.AlreadyExists,
		},
		{
			name: "duplicate username",
			req: &userpb.CreateUserRequest{
				Email:    "test2@example.com",
				Username: "testuser",
			},
			wantErr: true,
			errCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.CreateUser(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.User.Id)
			assert.Equal(t, tt.req.Email, resp.User.Email)
			assert.Equal(t, tt.req.Username, resp.User.Username)
			assert.Equal(t, tt.req.Roles, resp.User.Roles)
			assert.Equal(t, userpb.UserStatus_USER_STATUS_ACTIVE, resp.User.Status)
			assert.NotZero(t, resp.User.CreatedAt)
			assert.NotZero(t, resp.User.UpdatedAt)
			assert.Equal(t, tt.req.Metadata, resp.User.Metadata)
		})
	}
}

func TestGetUser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create a test user first
	createResp, err := service.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
	})
	require.NoError(t, err)
	userId := createResp.User.Id

	tests := []struct {
		name    string
		req     *userpb.GetUserRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful get user",
			req: &userpb.GetUserRequest{
				UserId: userId,
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: &userpb.GetUserRequest{
				UserId: "nonexistent",
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetUser(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, userId, resp.User.Id)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create a test user first
	createResp, err := service.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Profile: &userpb.UserProfile{
			FirstName: "Test",
			LastName:  "User",
		},
	})
	require.NoError(t, err)
	userId := createResp.User.Id

	tests := []struct {
		name           string
		req            *userpb.UpdateUserRequest
		wantErr        bool
		errCode        codes.Code
		fieldsToUpdate []string
	}{
		{
			name: "update all fields",
			req: &userpb.UpdateUserRequest{
				UserId: userId,
				User: &userpb.User{
					Email:    "updated@example.com",
					Username: "updateduser",
					Profile: &userpb.UserProfile{
						FirstName: "Updated",
						LastName:  "User",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update specific fields",
			req: &userpb.UpdateUserRequest{
				UserId: userId,
				User: &userpb.User{
					Email:    "specific@example.com",
					Username: "specificuser",
					Profile: &userpb.UserProfile{
						FirstName: "Specific",
						LastName:  "Update",
					},
				},
				FieldsToUpdate: []string{"email", "profile"},
			},
			wantErr:        false,
			fieldsToUpdate: []string{"email", "profile"},
		},
		{
			name: "user not found",
			req: &userpb.UpdateUserRequest{
				UserId: "nonexistent",
				User: &userpb.User{
					Email: "notfound@example.com",
				},
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.UpdateUser(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)

			if len(tt.fieldsToUpdate) > 0 {
				// Check only updated fields
				for _, field := range tt.fieldsToUpdate {
					switch field {
					case "email":
						assert.Equal(t, tt.req.User.Email, resp.User.Email)
					case "username":
						assert.Equal(t, tt.req.User.Username, resp.User.Username)
					case "profile":
						assert.Equal(t, tt.req.User.Profile, resp.User.Profile)
					}
				}
			} else {
				// Check all fields
				assert.Equal(t, tt.req.User.Email, resp.User.Email)
				assert.Equal(t, tt.req.User.Username, resp.User.Username)
				assert.Equal(t, tt.req.User.Profile, resp.User.Profile)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create a test user first
	createResp, err := service.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
	})
	require.NoError(t, err)
	userId := createResp.User.Id

	tests := []struct {
		name    string
		req     *userpb.DeleteUserRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "successful delete",
			req: &userpb.DeleteUserRequest{
				UserId: userId,
			},
			wantErr: false,
		},
		{
			name: "user not found",
			req: &userpb.DeleteUserRequest{
				UserId: "nonexistent",
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.DeleteUser(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Success)

			// Verify user is actually deleted
			_, err = service.GetUser(context.Background(), &userpb.GetUserRequest{UserId: tt.req.UserId})
			assert.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		})
	}
}

func TestListUsers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create test users
	users := []*userpb.CreateUserRequest{
		{
			Email:    "user1@example.com",
			Username: "user1",
		},
		{
			Email:    "user2@example.com",
			Username: "user2",
		},
		{
			Email:    "user3@example.com",
			Username: "user3",
		},
	}

	for _, user := range users {
		_, err := service.CreateUser(context.Background(), user)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		req           *userpb.ListUsersRequest
		wantErr       bool
		expectedCount int32
	}{
		{
			name: "list all users",
			req: &userpb.ListUsersRequest{
				Page:     0,
				PageSize: 10,
			},
			wantErr:       false,
			expectedCount: 3,
		},
		{
			name: "filter by status",
			req: &userpb.ListUsersRequest{
				Page:     0,
				PageSize: 10,
				Filters: map[string]string{
					"status": userpb.UserStatus_USER_STATUS_ACTIVE.String(),
				},
			},
			wantErr:       false,
			expectedCount: 3, // All users are active by default
		},
		{
			name: "pagination",
			req: &userpb.ListUsersRequest{
				Page:     0,
				PageSize: 2,
			},
			wantErr:       false,
			expectedCount: 3, // Total count should still be 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.ListUsers(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedCount, resp.TotalCount)

			if tt.req.PageSize > 0 {
				assert.LessOrEqual(t, len(resp.Users), int(tt.req.PageSize))
			}

			if len(tt.req.Filters) > 0 {
				for _, user := range resp.Users {
					if status, ok := tt.req.Filters["status"]; ok {
						assert.Equal(t, status, user.Status.String())
					}
				}
			}
		})
	}
}

func TestUpdateProfile(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create a test user first
	createResp, err := service.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
		Profile: &userpb.UserProfile{
			FirstName:   "Test",
			LastName:    "User",
			PhoneNumber: "+1234567890",
		},
	})
	require.NoError(t, err)
	userId := createResp.User.Id

	tests := []struct {
		name           string
		req            *userpb.UpdateProfileRequest
		wantErr        bool
		errCode        codes.Code
		fieldsToUpdate []string
	}{
		{
			name: "update all profile fields",
			req: &userpb.UpdateProfileRequest{
				UserId: userId,
				Profile: &userpb.UserProfile{
					FirstName:   "Updated",
					LastName:    "User",
					PhoneNumber: "+9876543210",
					AvatarUrl:   "https://example.com/avatar.jpg",
					Bio:         "Updated bio",
					Location:    "New Location",
					Timezone:    "UTC+1",
					Language:    "fr",
					CustomFields: map[string]string{
						"occupation": "developer",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update specific profile fields",
			req: &userpb.UpdateProfileRequest{
				UserId: userId,
				Profile: &userpb.UserProfile{
					FirstName: "Specific",
					LastName:  "Update",
					Bio:       "New bio",
				},
				FieldsToUpdate: []string{"first_name", "last_name", "bio"},
			},
			wantErr:        false,
			fieldsToUpdate: []string{"first_name", "last_name", "bio"},
		},
		{
			name: "user not found",
			req: &userpb.UpdateProfileRequest{
				UserId: "nonexistent",
				Profile: &userpb.UserProfile{
					FirstName: "NotFound",
					LastName:  "User",
				},
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.UpdateProfile(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.errCode, st.Code())
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)

			if len(tt.fieldsToUpdate) > 0 {
				// Check only updated fields
				for _, field := range tt.fieldsToUpdate {
					switch field {
					case "first_name":
						assert.Equal(t, tt.req.Profile.FirstName, resp.User.Profile.FirstName)
					case "last_name":
						assert.Equal(t, tt.req.Profile.LastName, resp.User.Profile.LastName)
					case "bio":
						assert.Equal(t, tt.req.Profile.Bio, resp.User.Profile.Bio)
					}
				}
			} else {
				// Check all fields
				assert.Equal(t, tt.req.Profile, resp.User.Profile)
			}
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewUserService(logger)

	// Create a test user first
	createResp, err := service.CreateUser(context.Background(), &userpb.CreateUserRequest{
		Email:    "test@example.com",
		Username: "testuser",
	})
	require.NoError(t, err)
	userId := createResp.User.Id

	tests := []struct {
		name    string
		req     *userpb.UpdatePasswordRequest
		wantErr bool
	}{
		{
			name: "successful password update",
			req: &userpb.UpdatePasswordRequest{
				UserId:          userId,
				CurrentPassword: "oldpassword",
				NewPassword:     "newpassword",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.UpdatePassword(context.Background(), tt.req)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp)
			assert.True(t, resp.Success)
			assert.NotZero(t, resp.UpdatedAt)
		})
	}
}
