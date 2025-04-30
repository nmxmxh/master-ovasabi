# API Documentation

## Overview

This documentation covers the API design, implementation, and usage guidelines for the OVASABI
platform.

## API Design Principles

1. **RESTful Design**

   ```go
   // Example from internal/api/handler.go
   type Handler struct {
       service Service
   }

   func (h *Handler) RegisterRoutes(r *mux.Router) {
       r.HandleFunc("/api/v1/users", h.CreateUser).Methods("POST")
       r.HandleFunc("/api/v1/users/{id}", h.GetUser).Methods("GET")
       r.HandleFunc("/api/v1/users/{id}", h.UpdateUser).Methods("PUT")
       r.HandleFunc("/api/v1/users/{id}", h.DeleteUser).Methods("DELETE")
   }
   ```

2. **gRPC Services**

   ```protobuf
   // Example from api/protos/user.proto
   service UserService {
       rpc CreateUser(CreateUserRequest) returns (User) {}
       rpc GetUser(GetUserRequest) returns (User) {}
       rpc UpdateUser(UpdateUserRequest) returns (User) {}
       rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty) {}
   }
   ```

## Authentication and Authorization

1. **JWT Authentication**

   ```go
   // Example from internal/auth/jwt.go
   type JWT struct {
       secret []byte
   }

   func (j *JWT) GenerateToken(user *User) (string, error) {
       claims := jwt.MapClaims{
           "user_id": user.ID,
           "exp":     time.Now().Add(24 * time.Hour).Unix(),
       }

       token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
       return token.SignedString(j.secret)
   }
   ```

2. **Role-Based Access**

   ```go
   // Example from internal/auth/authorizer.go
   type Authorizer struct {
       roles map[string][]string
   }

   func (a *Authorizer) HasPermission(user *User, resource string, action string) bool {
       roles, ok := a.roles[user.Role]
       if !ok {
           return false
       }

       for _, role := range roles {
           if role == resource+":"+action {
               return true
           }
       }

       return false
   }
   ```

## Request/Response Handling

1. **Request Validation**

   ```go
   // Example from internal/api/validator.go
   type Validator struct {
       validate *validator.Validate
   }

   func (v *Validator) ValidateRequest(req interface{}) error {
       if err := v.validate.Struct(req); err != nil {
           return fmt.Errorf("validation failed: %w", err)
       }
       return nil
   }
   ```

2. **Response Formatting**

   ```go
   // Example from internal/api/response.go
   type Response struct {
       Status  string      `json:"status"`
       Data    interface{} `json:"data,omitempty"`
       Error   string      `json:"error,omitempty"`
       Message string      `json:"message,omitempty"`
   }

   func NewSuccessResponse(data interface{}) *Response {
       return &Response{
           Status: "success",
           Data:   data,
       }
   }
   ```

## Error Handling

1. **Error Types**

   ```go
   // Example from pkg/errors/errors.go
   type Error struct {
       Code    string `json:"code"`
       Message string `json:"message"`
       Status  int    `json:"-"`
   }

   func (e *Error) Error() string {
       return e.Message
   }
   ```

2. **Error Responses**

   ```go
   // Example from internal/api/error.go
   func (h *Handler) handleError(w http.ResponseWriter, err error) {
       var apiErr *Error
       if errors.As(err, &apiErr) {
           w.WriteHeader(apiErr.Status)
           pkg.NewEncoder(w).Encode(apiErr)
           return
       }

       w.WriteHeader(http.StatusInternalServerError)
       pkg.NewEncoder(w).Encode(&Error{
           Code:    "internal_error",
           Message: "An internal error occurred",
       })
   }
   ```

## Rate Limiting

1. **Rate Limiter**

   ```go
   // Example from internal/api/limiter.go
   type RateLimiter struct {
       limiter *rate.Limiter
   }

   func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           if !rl.limiter.Allow() {
               http.Error(w, "Too many requests", http.StatusTooManyRequests)
               return
           }
           next.ServeHTTP(w, r)
       })
   }
   ```

2. **IP-Based Limiting**

   ```go
   // Example from internal/api/ip_limiter.go
   type IPLimiter struct {
       limiters map[string]*rate.Limiter
       mu       sync.RWMutex
   }

   func (il *IPLimiter) getLimiter(ip string) *rate.Limiter {
       il.mu.Lock()
       defer il.mu.Unlock()

       limiter, exists := il.limiters[ip]
       if !exists {
           limiter = rate.NewLimiter(rate.Limit(10), 20)
           il.limiters[ip] = limiter
       }

       return limiter
   }
   ```

## API Versioning

1. **Version Header**

   ```go
   // Example from internal/api/version.go
   const (
       APIVersion = "v1"
       VersionHeader = "X-API-Version"
   )

   func (h *Handler) checkVersion(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           version := r.Header.Get(VersionHeader)
           if version != APIVersion {
               http.Error(w, "Unsupported API version", http.StatusBadRequest)
               return
           }
           next.ServeHTTP(w, r)
       })
   }
   ```

2. **URL Versioning**

   ```go
   // Example from internal/api/router.go
   func (h *Handler) setupRoutes() *mux.Router {
       r := mux.NewRouter()

       // Version 1 routes
       v1 := r.PathPrefix("/api/v1").Subrouter()
       h.registerV1Routes(v1)

       // Version 2 routes
       v2 := r.PathPrefix("/api/v2").Subrouter()
       h.registerV2Routes(v2)

       return r
   }
   ```

## Documentation

1. **OpenAPI/Swagger**

   ```yaml
   # Example from api/swagger.yaml
   openapi: 3.0.0
   info:
     title: OVASABI API
     version: 1.0.0

   paths:
     /api/v1/users:
       post:
         summary: Create a new user
         requestBody:
           required: true
           content:
             application/json:
               schema:
                 $ref: '#/components/schemas/User'
   ```

2. **gRPC Documentation**

   ```protobuf
   // Example from api/protos/user.proto
   message User {
       // User ID
       string id = 1;
       // User email
       string email = 2;
       // User name
       string name = 3;
   }
   ```

## Testing

1. **API Tests**

   ```go
   // Example from test/api/user_test.go
   func TestCreateUser(t *testing.T) {
       req := &CreateUserRequest{
           Email: "test@example.com",
           Name:  "Test User",
       }

       resp, err := client.CreateUser(context.Background(), req)
       if err != nil {
           t.Fatalf("CreateUser failed: %v", err)
       }

       if resp.Email != req.Email {
           t.Errorf("got %v, want %v", resp.Email, req.Email)
       }
   }
   ```

2. **Integration Tests**

   ```go
   // Example from test/integration/api_test.go
   func TestAPI(t *testing.T) {
       // Setup test server
       srv := setupTestServer(t)
       defer srv.Close()

       // Run tests
       client := newTestClient(srv.URL)

       // Test endpoints
       testCreateUser(t, client)
       testGetUser(t, client)
       testUpdateUser(t, client)
       testDeleteUser(t, client)
   }
   ```
