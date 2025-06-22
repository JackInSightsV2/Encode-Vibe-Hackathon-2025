# Testing Framework & Coverage Enhancement - Refined Implementation Cycles

## Overview
Break down comprehensive testing framework into 5 manageable cycles, each delivering testable functionality with clear outcomes.

---

## **Cycle 19A: Backend Testing Foundation**
**Duration:** 5-6 hours | **Priority:** Critical

### Prerequisites
- Go development environment ready
- Understanding of Go testing patterns and table-driven tests
- Knowledge of test doubles (mocks, stubs, fakes)

### Implementation Tasks
- [ ] Set up Go testing framework with required dependencies
- [ ] Create test utility functions and fixtures
- [ ] Implement unit tests for core business logic
- [ ] Add integration test setup with test containers
- [ ] Configure test coverage reporting
- [ ] Create test data factories and builders

### Code Deliverables
```go
// backend/test/setup.go
package test

import (
    "context"
    "database/sql"
    "testing"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/modules/redis"
    _ "github.com/lib/pq"
)

type TestSuite struct {
    DB            *sql.DB
    Redis         *redis.Client
    PostgresContainer *postgres.PostgresContainer
    RedisContainer    *redis.RedisContainer
    Ctx           context.Context
    CleanupFuncs  []func()
}

func NewTestSuite(t *testing.T) *TestSuite {
    ctx := context.Background()
    
    // Start PostgreSQL container
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("qt1_test"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second)),
    )
    require.NoError(t, err)
    
    // Start Redis container  
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)
    
    // Connect to PostgreSQL
    connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)
    
    // Connect to Redis
    redisHost, err := redisContainer.Host(ctx)
    require.NoError(t, err)
    redisPort, err := redisContainer.MappedPort(ctx, "6379")
    require.NoError(t, err)
    
    rdb := redis.NewClient(&redis.Options{
        Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
    })
    
    suite := &TestSuite{
        DB:                db,
        Redis:             rdb,
        PostgresContainer: postgresContainer,
        RedisContainer:    redisContainer,
        Ctx:               ctx,
        CleanupFuncs:      []func(){},
    }
    
    // Run migrations
    suite.RunMigrations(t)
    
    return suite
}

func (ts *TestSuite) Cleanup(t *testing.T) {
    for _, cleanup := range ts.CleanupFuncs {
        cleanup()
    }
    
    if ts.DB != nil {
        ts.DB.Close()
    }
    
    if ts.Redis != nil {
        ts.Redis.Close()
    }
    
    if ts.PostgresContainer != nil {
        require.NoError(t, ts.PostgresContainer.Terminate(ts.Ctx))
    }
    
    if ts.RedisContainer != nil {
        require.NoError(t, ts.RedisContainer.Terminate(ts.Ctx))
    }
}

func (ts *TestSuite) RunMigrations(t *testing.T) {
    // Run database migrations for testing
    migrations := []string{
        `CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            username VARCHAR(100) UNIQUE NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        )`,
        `CREATE TABLE IF NOT EXISTS chat_sessions (
            id SERIAL PRIMARY KEY,
            user_id INTEGER REFERENCES users(id),
            session_id VARCHAR(100) NOT NULL,
            created_at TIMESTAMP DEFAULT NOW()
        )`,
    }
    
    for _, migration := range migrations {
        _, err := ts.DB.ExecContext(ts.Ctx, migration)
        require.NoError(t, err)
    }
}

// Test data factories
type UserFactory struct {
    db *sql.DB
}

func NewUserFactory(db *sql.DB) *UserFactory {
    return &UserFactory{db: db}
}

func (uf *UserFactory) Create(options ...func(*User)) *User {
    user := &User{
        Username: fmt.Sprintf("testuser_%d", time.Now().UnixNano()),
        Email:    fmt.Sprintf("test_%d@example.com", time.Now().UnixNano()),
    }
    
    for _, option := range options {
        option(user)
    }
    
    query := `INSERT INTO users (username, email) VALUES ($1, $2) RETURNING id, created_at`
    err := uf.db.QueryRow(query, user.Username, user.Email).Scan(&user.ID, &user.CreatedAt)
    if err != nil {
        panic(err)
    }
    
    return user
}

func WithUsername(username string) func(*User) {
    return func(u *User) {
        u.Username = username
    }
}

func WithEmail(email string) func(*User) {
    return func(u *User) {
        u.Email = email
    }
}
```

```go
// backend/middleware/proxy_test.go
package middleware

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
    "github.com/qt1-middleware/backend/test"
)

func TestProxy_HandleChat(t *testing.T) {
    tests := []struct {
        name           string
        request        ChatRequest
        setupMocks     func(*MockAIProvider, *MockModerator)
        expectedStatus int
        expectedError  string
        validateResponse func(*testing.T, *ChatResponse)
    }{
        {
            name: "successful_chat_request",
            request: ChatRequest{
                UserID:    "user123",
                SessionID: "session456", 
                Message:   "Hello world",
                Provider:  "openai",
                Model:     "gpt-3.5-turbo",
            },
            setupMocks: func(provider *MockAIProvider, moderator *MockModerator) {
                moderator.On("CheckContent", mock.Anything, "Hello world").
                    Return(&ModerationResult{Passed: true, Score: 0.1}, nil)
                    
                provider.On("GenerateResponse", mock.Anything, mock.MatchedBy(func(req *AIRequest) bool {
                    return req.Message == "Hello world" && req.Model == "gpt-3.5-turbo"
                })).Return(&AIResponse{
                    Content:      "Hello! How can I help you today?",
                    TokensUsed:   25,
                    CostEstimate: 0.001,
                    ResponseTime: 500 * time.Millisecond,
                }, nil)
            },
            expectedStatus: http.StatusOK,
            validateResponse: func(t *testing.T, resp *ChatResponse) {
                assert.Equal(t, "Hello! How can I help you today?", resp.Response)
                assert.Equal(t, "openai", resp.Provider)
                assert.Equal(t, "gpt-3.5-turbo", resp.Model)
                assert.True(t, resp.ModerationPassed)
                assert.Equal(t, 25, resp.TokensUsed)
                assert.InDelta(t, 0.001, resp.CostEstimate, 0.0001)
            },
        },
        {
            name: "moderation_failure",
            request: ChatRequest{
                UserID:    "user123",
                SessionID: "session456",
                Message:   "inappropriate content here",
            },
            setupMocks: func(provider *MockAIProvider, moderator *MockModerator) {
                moderator.On("CheckContent", mock.Anything, "inappropriate content here").
                    Return(&ModerationResult{Passed: false, Score: 0.9, Reason: "inappropriate"}, nil)
            },
            expectedStatus: http.StatusBadRequest,
            expectedError:  "Content moderation failed",
        },
        {
            name: "provider_error",
            request: ChatRequest{
                UserID:    "user123", 
                SessionID: "session456",
                Message:   "Hello world",
            },
            setupMocks: func(provider *MockAIProvider, moderator *MockModerator) {
                moderator.On("CheckContent", mock.Anything, "Hello world").
                    Return(&ModerationResult{Passed: true, Score: 0.1}, nil)
                    
                provider.On("GenerateResponse", mock.Anything, mock.Anything).
                    Return(nil, errors.New("provider unavailable"))
            },
            expectedStatus: http.StatusServiceUnavailable,
            expectedError:  "AI provider unavailable",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            suite := test.NewTestSuite(t)
            defer suite.Cleanup(t)
            
            mockProvider := &MockAIProvider{}
            mockModerator := &MockModerator{}
            
            if tt.setupMocks != nil {
                tt.setupMocks(mockProvider, mockModerator)
            }
            
            proxy := NewProxy(&ProxyConfig{
                DB:        suite.DB,
                Redis:     suite.Redis,
                Provider:  mockProvider,
                Moderator: mockModerator,
            })
            
            // Create request
            reqBody, err := json.Marshal(tt.request)
            require.NoError(t, err)
            
            req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(reqBody))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", "Bearer test-api-key")
            
            rr := httptest.NewRecorder()
            
            // Execute
            proxy.HandleChat(rr, req)
            
            // Assert
            assert.Equal(t, tt.expectedStatus, rr.Code)
            
            if tt.expectedError != "" {
                var errorResp ErrorResponse
                err := json.Unmarshal(rr.Body.Bytes(), &errorResp)
                require.NoError(t, err)
                assert.Contains(t, errorResp.Detail, tt.expectedError)
            }
            
            if tt.validateResponse != nil {
                var chatResp ChatResponse
                err := json.Unmarshal(rr.Body.Bytes(), &chatResp)
                require.NoError(t, err)
                tt.validateResponse(t, &chatResp)
            }
            
            // Verify mocks
            mockProvider.AssertExpectations(t)
            mockModerator.AssertExpectations(t)
        })
    }
}

func TestProxy_RateLimiting(t *testing.T) {
    suite := test.NewTestSuite(t)
    defer suite.Cleanup(t)
    
    proxy := NewProxy(&ProxyConfig{
        DB:    suite.DB,
        Redis: suite.Redis,
        RateLimit: &RateLimitConfig{
            RequestsPerMinute: 2,
            BurstSize:        1,
        },
    })
    
    // First request should succeed
    req1 := createChatRequest(t, ChatRequest{
        UserID: "rate_test_user", SessionID: "session1", Message: "Hello 1",
    })
    rr1 := httptest.NewRecorder()
    proxy.HandleChat(rr1, req1)
    assert.Equal(t, http.StatusOK, rr1.Code)
    
    // Second request should succeed (burst)
    req2 := createChatRequest(t, ChatRequest{
        UserID: "rate_test_user", SessionID: "session1", Message: "Hello 2",
    })
    rr2 := httptest.NewRecorder()
    proxy.HandleChat(rr2, req2)
    assert.Equal(t, http.StatusOK, rr2.Code)
    
    // Third request should be rate limited
    req3 := createChatRequest(t, ChatRequest{
        UserID: "rate_test_user", SessionID: "session1", Message: "Hello 3",
    })
    rr3 := httptest.NewRecorder()
    proxy.HandleChat(rr3, req3)
    assert.Equal(t, http.StatusTooManyRequests, rr3.Code)
}
```

### Testing Requirements
- [ ] Test all unit test suites pass with >90% coverage
- [ ] Test integration tests with real databases work
- [ ] Test mock objects behave correctly
- [ ] Test data factories generate valid test data
- [ ] Test parallel test execution works without conflicts

### Acceptance Criteria
- [ ] Unit test coverage >90% for core business logic
- [ ] Integration tests validate database interactions
- [ ] Test suite runs in <2 minutes on CI
- [ ] All tests are deterministic and non-flaky
- [ ] Test data cleanup prevents test pollution
- [ ] Mock verification catches integration issues

### Risk Mitigation
- Use test containers for database isolation
- Implement proper test cleanup and teardown
- Add test data randomization to prevent flakiness

---

## **Cycle 19B: Frontend Component Testing**
**Duration:** 4-5 hours | **Priority:** High

### Prerequisites
- Cycle 19A completed
- React Testing Library and Jest configured
- Understanding of component testing patterns

### Implementation Tasks
- [ ] Set up React Testing Library with Jest
- [ ] Create component test utilities and helpers
- [ ] Implement unit tests for all React components
- [ ] Add user interaction testing
- [ ] Create visual regression testing setup
- [ ] Add accessibility testing

### Code Deliverables
```json
// frontend/package.json - Testing dependencies
{
  "devDependencies": {
    "@testing-library/react": "^13.4.0",
    "@testing-library/jest-dom": "^5.16.5",
    "@testing-library/user-event": "^14.4.3",
    "jest": "^29.5.0",
    "jest-environment-jsdom": "^29.5.0",
    "msw": "^1.2.1",
    "@testing-library/react-hooks": "^8.0.1",
    "jest-axe": "^7.0.1",
    "@percy/cli": "^1.6.1",
    "@percy/jest-puppeteer": "^2.0.1"
  }
}
```

```javascript
// frontend/src/test/test-utils.tsx
import React from 'react';
import { render as rtlRender } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from '../contexts/AuthContext';
import { ThemeProvider } from '../contexts/ThemeContext';

// Mock user for testing
export const mockUser = {
  id: 'test-user-123',
  username: 'testuser',
  email: 'test@example.com',
  role: 'admin'
};

// Mock auth context value
export const mockAuthValue = {
  user: mockUser,
  login: jest.fn(),
  logout: jest.fn(),
  isAuthenticated: true,
  isLoading: false
};

// Custom render function with providers
function render(ui, {
  route = '/',
  user = mockUser,
  authValue = mockAuthValue,
  queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  }),
  ...renderOptions
} = {}) {
  
  // Navigate to route
  window.history.pushState({}, 'Test page', route);
  
  function Wrapper({ children }) {
    return (
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <AuthProvider value={authValue}>
            <ThemeProvider>
              {children}
            </ThemeProvider>
          </AuthProvider>
        </QueryClientProvider>
      </BrowserRouter>
    );
  }
  
  const renderResult = rtlRender(ui, { wrapper: Wrapper, ...renderOptions });
  
  return {
    user: userEvent.setup(),
    ...renderResult
  };
}

// Mock API responses
export const mockApiResponse = (data, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  json: () => Promise.resolve(data),
  text: () => Promise.resolve(JSON.stringify(data))
});

// Wait for loading to complete
export const waitForLoadingToFinish = () => 
  waitForElementToBeRemoved(() => screen.queryByTestId('loading-spinner'));

// Accessibility testing helper
export const axeTest = async (container) => {
  const { toHaveNoViolations } = require('jest-axe');
  expect.extend(toHaveNoViolations);
  
  const results = await axe(container);
  expect(results).toHaveNoViolations();
};

// Re-export everything
export * from '@testing-library/react';
export { render, userEvent };
```

```javascript
// frontend/src/components/__tests__/Dashboard.test.tsx
import React from 'react';
import { screen, waitFor } from '@testing-library/react';
import { render, mockApiResponse, axeTest } from '../../test/test-utils';
import { Dashboard } from '../Dashboard';
import { rest } from 'msw';
import { setupServer } from 'msw/node';

// Mock data
const mockSystemStatus = {
  status: 'healthy',
  uptime: '24h 15m',
  requests_today: 1234,
  errors_today: 5,
  providers: {
    openai: { status: 'healthy', response_time: 120 },
    anthropic: { status: 'healthy', response_time: 95 },
    google: { status: 'degraded', response_time: 250 }
  }
};

const mockMetrics = {
  total_requests: 45678,
  avg_response_time: 145,
  error_rate: 0.4,
  cost_today: 23.45
};

// Setup MSW server
const server = setupServer(
  rest.get('/api/system/status', (req, res, ctx) => {
    return res(ctx.json(mockSystemStatus));
  }),
  rest.get('/api/metrics/overview', (req, res, ctx) => {
    return res(ctx.json(mockMetrics));
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('Dashboard', () => {
  it('renders system status correctly', async () => {
    render(<Dashboard />);
    
    // Check loading state
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
    
    // Wait for data to load
    await waitFor(() => {
      expect(screen.getByText('System Online')).toBeInTheDocument();
    });
    
    // Check system status elements
    expect(screen.getByText('24h 15m')).toBeInTheDocument();
    expect(screen.getByText('1234')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });
  
  it('displays provider status correctly', async () => {
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('OpenAI')).toBeInTheDocument();
    });
    
    // Check provider statuses
    expect(screen.getByTestId('provider-openai')).toHaveClass('status-healthy');
    expect(screen.getByTestId('provider-google')).toHaveClass('status-degraded');
    
    // Check response times
    expect(screen.getByText('120ms')).toBeInTheDocument();
    expect(screen.getByText('250ms')).toBeInTheDocument();
  });
  
  it('handles API errors gracefully', async () => {
    // Mock API error
    server.use(
      rest.get('/api/system/status', (req, res, ctx) => {
        return res(ctx.status(500), ctx.json({ error: 'Internal server error' }));
      })
    );
    
    render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText(/error loading dashboard/i)).toBeInTheDocument();
    });
    
    // Check retry button appears
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument();
  });
  
  it('refreshes data when refresh button clicked', async () => {
    const { user } = render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('1234')).toBeInTheDocument();
    });
    
    // Mock updated data
    server.use(
      rest.get('/api/system/status', (req, res, ctx) => {
        return res(ctx.json({
          ...mockSystemStatus,
          requests_today: 1300
        }));
      })
    );
    
    // Click refresh button
    const refreshButton = screen.getByRole('button', { name: /refresh/i });
    await user.click(refreshButton);
    
    // Check updated data appears
    await waitFor(() => {
      expect(screen.getByText('1300')).toBeInTheDocument();
    });
  });
  
  it('is accessible', async () => {
    const { container } = render(<Dashboard />);
    
    await waitFor(() => {
      expect(screen.getByText('System Online')).toBeInTheDocument();
    });
    
    await axeTest(container);
  });
  
  it('handles responsive design', () => {
    // Mock different viewport sizes
    global.innerWidth = 320;
    global.dispatchEvent(new Event('resize'));
    
    render(<Dashboard />);
    
    // Check mobile-specific elements
    expect(screen.getByTestId('mobile-dashboard')).toBeInTheDocument();
    
    // Reset viewport
    global.innerWidth = 1024;
    global.dispatchEvent(new Event('resize'));
  });
});
```

```javascript
// frontend/src/hooks/__tests__/useSystemStatus.test.ts
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useSystemStatus } from '../useSystemStatus';
import { rest } from 'msw';
import { setupServer } from 'msw/node';

const mockData = {
  status: 'healthy',
  uptime: '1h 30m',
  requests_today: 500
};

const server = setupServer(
  rest.get('/api/system/status', (req, res, ctx) => {
    return res(ctx.json(mockData));
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false }
    }
  });
  
  return ({ children }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('useSystemStatus', () => {
  it('fetches system status successfully', async () => {
    const { result } = renderHook(() => useSystemStatus(), {
      wrapper: createWrapper()
    });
    
    expect(result.current.isLoading).toBe(true);
    
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
    
    expect(result.current.data).toEqual(mockData);
    expect(result.current.error).toBeNull();
  });
  
  it('handles error states', async () => {
    server.use(
      rest.get('/api/system/status', (req, res, ctx) => {
        return res(ctx.status(500));
      })
    );
    
    const { result } = renderHook(() => useSystemStatus(), {
      wrapper: createWrapper()
    });
    
    await waitFor(() => {
      expect(result.current.error).toBeTruthy();
    });
    
    expect(result.current.data).toBeUndefined();
  });
  
  it('refetches data correctly', async () => {
    const { result } = renderHook(() => useSystemStatus(), {
      wrapper: createWrapper()
    });
    
    await waitFor(() => {
      expect(result.current.data).toEqual(mockData);
    });
    
    // Trigger refetch
    result.current.refetch();
    
    await waitFor(() => {
      expect(result.current.isFetching).toBe(true);
    });
    
    await waitFor(() => {
      expect(result.current.isFetching).toBe(false);
    });
  });
});
```

### Testing Requirements
- [ ] Test all React components render correctly
- [ ] Test user interactions and event handlers
- [ ] Test accessibility compliance
- [ ] Test responsive behavior
- [ ] Test error states and loading states

### Acceptance Criteria
- [ ] Component test coverage >85%
- [ ] All user interactions tested with realistic scenarios
- [ ] Accessibility tests pass with zero violations
- [ ] Visual regression tests detect UI changes
- [ ] Tests run in <30 seconds
- [ ] No test flakiness or random failures

### Risk Mitigation
- Use MSW for consistent API mocking
- Test with realistic user interactions
- Implement proper cleanup between tests

---

## **Cycle 19C: End-to-End Testing with Playwright**
**Duration:** 5-6 hours | **Priority:** Medium

### Prerequisites
- Cycles 19A and 19B completed
- Playwright installed and configured
- Understanding of E2E testing patterns

### Implementation Tasks
- [ ] Set up Playwright for cross-browser testing
- [ ] Create page object models for maintainable tests
- [ ] Implement critical user journey tests
- [ ] Add visual regression testing
- [ ] Create test data management system
- [ ] Add performance testing capabilities

### Code Deliverables
```javascript
// e2e/playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 30 * 1000,
  expect: {
    timeout: 5000
  },
  
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  
  reporter: [
    ['html'],
    ['junit', { outputFile: 'results.xml' }],
    ['github']
  ],
  
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },

  projects: [
    // Setup project
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/
    },
    
    // Desktop browsers
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      dependencies: ['setup']
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
      dependencies: ['setup']
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
      dependencies: ['setup']
    },
    
    // Mobile devices
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
      dependencies: ['setup']
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
      dependencies: ['setup']
    }
  ],

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000
  }
});
```

```typescript
// e2e/pages/BasePage.ts
import { Page, Locator } from '@playwright/test';

export abstract class BasePage {
  protected page: Page;
  protected url: string;

  constructor(page: Page, url: string) {
    this.page = page;
    this.url = url;
  }

  async goto() {
    await this.page.goto(this.url);
    await this.waitForPageLoad();
  }

  async waitForPageLoad() {
    await this.page.waitForLoadState('networkidle');
  }

  // Common elements
  get navigationMenu(): Locator {
    return this.page.getByTestId('navigation-menu');
  }

  get loadingSpinner(): Locator {
    return this.page.getByTestId('loading-spinner');
  }

  get errorMessage(): Locator {
    return this.page.getByTestId('error-message');
  }

  // Common actions
  async waitForLoading() {
    await this.loadingSpinner.waitFor({ state: 'hidden', timeout: 10000 });
  }

  async takeScreenshot(name: string) {
    await this.page.screenshot({ 
      path: `screenshots/${name}.png`,
      fullPage: true 
    });
  }

  async expectPageTitle(title: string) {
    await expect(this.page).toHaveTitle(title);
  }
}
```

```typescript
// e2e/pages/DashboardPage.ts
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

export class DashboardPage extends BasePage {
  constructor(page: Page) {
    super(page, '/dashboard');
  }

  // Locators
  get systemStatusCard(): Locator {
    return this.page.getByTestId('system-status-card');
  }

  get metricsCard(): Locator {
    return this.page.getByTestId('metrics-card');
  }

  get providersCard(): Locator {
    return this.page.getByTestId('providers-card');
  }

  get refreshButton(): Locator {
    return this.page.getByRole('button', { name: /refresh/i });
  }

  getProviderStatus(provider: string): Locator {
    return this.page.getByTestId(`provider-${provider}`);
  }

  // Actions
  async refreshDashboard() {
    await this.refreshButton.click();
    await this.waitForLoading();
  }

  async waitForDashboardToLoad() {
    await this.systemStatusCard.waitFor();
    await this.metricsCard.waitFor();
    await this.providersCard.waitFor();
  }

  // Assertions
  async expectSystemStatus(status: 'healthy' | 'degraded' | 'unhealthy') {
    const statusElement = this.systemStatusCard.getByTestId('status-indicator');
    await expect(statusElement).toHaveClass(new RegExp(`status-${status}`));
  }

  async expectMetricValue(metric: string, value: string) {
    const metricElement = this.metricsCard.getByTestId(`metric-${metric}`);
    await expect(metricElement).toHaveText(value);
  }

  async expectProviderStatus(provider: string, status: string) {
    const providerElement = this.getProviderStatus(provider);
    await expect(providerElement).toContainText(status);
  }
}
```

```typescript
// e2e/tests/dashboard.spec.ts
import { test, expect } from '@playwright/test';
import { DashboardPage } from '../pages/DashboardPage';
import { LoginPage } from '../pages/LoginPage';

test.describe('Dashboard', () => {
  let dashboardPage: DashboardPage;

  test.beforeEach(async ({ page }) => {
    // Login before each test
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login('admin@qt1.com', 'password123');
    
    dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();
  });

  test('displays system status correctly', async () => {
    await dashboardPage.waitForDashboardToLoad();
    
    // Check that dashboard loads with correct elements
    await expect(dashboardPage.systemStatusCard).toBeVisible();
    await expect(dashboardPage.metricsCard).toBeVisible();
    await expect(dashboardPage.providersCard).toBeVisible();
    
    // Verify system status
    await dashboardPage.expectSystemStatus('healthy');
  });

  test('shows real-time metrics', async () => {
    await dashboardPage.waitForDashboardToLoad();
    
    // Check metrics are displayed
    await expect(dashboardPage.metricsCard.getByText(/total requests/i)).toBeVisible();
    await expect(dashboardPage.metricsCard.getByText(/avg response time/i)).toBeVisible();
    await expect(dashboardPage.metricsCard.getByText(/error rate/i)).toBeVisible();
  });

  test('refreshes data when refresh button clicked', async ({ page }) => {
    await dashboardPage.waitForDashboardToLoad();
    
    // Get initial request count
    const initialRequests = await dashboardPage.metricsCard
      .getByTestId('metric-total-requests')
      .textContent();
    
    // Click refresh and wait for update
    await dashboardPage.refreshButton.click();
    await dashboardPage.waitForLoading();
    
    // Verify data refreshed (this would need backend mock or test data)
    await expect(dashboardPage.metricsCard.getByTestId('metric-total-requests'))
      .not.toHaveText(initialRequests || '');
  });

  test('handles provider status correctly', async () => {
    await dashboardPage.waitForDashboardToLoad();
    
    // Check provider statuses
    await dashboardPage.expectProviderStatus('openai', 'Healthy');
    await dashboardPage.expectProviderStatus('anthropic', 'Healthy');
    
    // Check response times are displayed
    await expect(dashboardPage.getProviderStatus('openai'))
      .toContainText(/\d+ms/);
  });

  test('responsive design on mobile', async ({ page, isMobile }) => {
    test.skip(!isMobile, 'Mobile-specific test');
    
    await dashboardPage.waitForDashboardToLoad();
    
    // Check mobile navigation
    const mobileMenu = page.getByTestId('mobile-menu-button');
    await expect(mobileMenu).toBeVisible();
    
    // Check cards stack vertically on mobile
    const cards = page.getByTestId('dashboard-cards');
    await expect(cards).toHaveClass(/flex-col/);
  });

  test('visual regression', async ({ page }) => {
    await dashboardPage.waitForDashboardToLoad();
    
    // Take screenshot for visual regression testing
    await expect(page).toHaveScreenshot('dashboard-full.png');
    
    // Test individual components
    await expect(dashboardPage.systemStatusCard).toHaveScreenshot('system-status-card.png');
    await expect(dashboardPage.metricsCard).toHaveScreenshot('metrics-card.png');
  });
});
```

```typescript
// e2e/tests/chat-flow.spec.ts
import { test, expect } from '@playwright/test';
import { ChatPage } from '../pages/ChatPage';
import { DashboardPage } from '../pages/DashboardPage';

test.describe('Chat Flow', () => {
  test('complete chat workflow', async ({ page }) => {
    // Navigate to chat interface
    const chatPage = new ChatPage(page);
    await chatPage.goto();
    
    // Send a message
    await chatPage.sendMessage('Hello, how are you?');
    
    // Wait for response
    await chatPage.waitForResponse();
    
    // Verify response appears
    await expect(chatPage.getLastMessage()).toContainText(/hello/i);
    
    // Check that metrics update
    const dashboardPage = new DashboardPage(page);
    await dashboardPage.goto();
    await dashboardPage.waitForDashboardToLoad();
    
    // Verify request count increased
    await expect(dashboardPage.metricsCard.getByTestId('metric-total-requests'))
      .not.toHaveText('0');
  });

  test('handles moderation correctly', async ({ page }) => {
    const chatPage = new ChatPage(page);
    await chatPage.goto();
    
    // Send inappropriate content
    await chatPage.sendMessage('inappropriate test content');
    
    // Should see moderation warning
    await expect(chatPage.getModerationWarning()).toBeVisible();
    await expect(chatPage.getModerationWarning())
      .toContainText(/content moderation/i);
  });
});
```

### Testing Requirements
- [ ] Test critical user journeys end-to-end
- [ ] Test cross-browser compatibility
- [ ] Test mobile responsiveness
- [ ] Test visual regression detection
- [ ] Test performance under load

### Acceptance Criteria
- [ ] All critical user flows complete successfully
- [ ] Tests pass consistently across all target browsers
- [ ] Visual regression tests detect UI changes
- [ ] Mobile tests verify responsive behavior
- [ ] E2E test suite completes in <15 minutes
- [ ] Test failure rate <1% due to flakiness

### Risk Mitigation
- Use page object model for maintainability
- Implement proper wait strategies
- Add retry logic for flaky tests

---

## **Cycle 19D: Performance and Load Testing**
**Duration:** 4-5 hours | **Priority:** Medium

### Prerequisites
- Previous cycles completed
- Understanding of performance testing concepts
- Knowledge of load testing tools

### Implementation Tasks
- [ ] Set up Go benchmarking for backend performance
- [ ] Create load testing with artillery or k6
- [ ] Implement memory and CPU profiling
- [ ] Add database performance testing
- [ ] Create stress testing scenarios
- [ ] Set up performance monitoring

### Code Deliverables
```go
// backend/middleware/proxy_bench_test.go
package middleware

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

func BenchmarkProxy_HandleChat(b *testing.B) {
    // Setup
    proxy := setupBenchmarkProxy(b)
    request := ChatRequest{
        UserID:    "bench_user",
        SessionID: "bench_session",
        Message:   "Hello world benchmark test",
        Provider:  "openai",
        Model:     "gpt-3.5-turbo",
    }
    
    reqBody, _ := json.Marshal(request)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := httptest.NewRequest(http.MethodPost, "/chat", bytes.NewReader(reqBody))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", "Bearer test-key")
            
            rr := httptest.NewRecorder()
            proxy.HandleChat(rr, req)
            
            if rr.Code != http.StatusOK {
                b.Errorf("Expected 200, got %d", rr.Code)
            }
        }
    })
}

func BenchmarkCache_Operations(b *testing.B) {
    cache := setupBenchmarkCache(b)
    ctx := context.Background()
    
    testData := make([][]byte, 1000)
    for i := range testData {
        testData[i] = make([]byte, 1024) // 1KB each
    }
    
    b.Run("Set", func(b *testing.B) {
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            key := fmt.Sprintf("bench_key_%d", i%1000)
            cache.Set(ctx, key, testData[i%1000], time.Hour)
        }
    })
    
    b.Run("Get", func(b *testing.B) {
        // Pre-populate cache
        for i := 0; i < 1000; i++ {
            key := fmt.Sprintf("bench_key_%d", i)
            cache.Set(ctx, key, testData[i], time.Hour)
        }
        
        b.ResetTimer()
        b.RunParallel(func(pb *testing.PB) {
            for pb.Next() {
                key := fmt.Sprintf("bench_key_%d", rand.Intn(1000))
                _, _ = cache.Get(ctx, key)
            }
        })
    })
}

func BenchmarkDatabase_Queries(b *testing.B) {
    db := setupBenchmarkDatabase(b)
    defer db.Close()
    
    b.Run("SimpleSelect", func(b *testing.B) {
        query := "SELECT id, username, email FROM users WHERE id = $1"
        
        b.ResetTimer()
        b.RunParallel(func(pb *testing.PB) {
            for pb.Next() {
                userID := rand.Intn(10000) + 1
                rows, err := db.Query(query, userID)
                if err != nil {
                    b.Error(err)
                }
                rows.Close()
            }
        })
    })
    
    b.Run("ComplexJoin", func(b *testing.B) {
        query := `
            SELECT u.id, u.username, cs.session_id, COUNT(cr.id) as request_count
            FROM users u
            LEFT JOIN chat_sessions cs ON u.id = cs.user_id
            LEFT JOIN chat_requests cr ON cs.id = cr.session_id
            WHERE u.created_at > $1
            GROUP BY u.id, u.username, cs.session_id
            LIMIT 100
        `
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            since := time.Now().Add(-24 * time.Hour)
            rows, err := db.Query(query, since)
            if err != nil {
                b.Error(err)
            }
            rows.Close()
        }
    })
}
```

```javascript
// load-tests/artillery.yml
config:
  target: 'http://localhost:8080'
  phases:
    - duration: 60
      arrivalRate: 10
      name: "Warm up"
    - duration: 120
      arrivalRate: 50
      name: "Ramp up load"
    - duration: 300
      arrivalRate: 100
      name: "Sustained load"
    - duration: 60
      arrivalRate: 200
      name: "Spike test"
  
  variables:
    apiKey: "test-api-key-123"
  
  plugins:
    metrics-by-endpoint:
      useOnlyRequestNames: true

scenarios:
  - name: "Chat API Load Test"
    weight: 80
    flow:
      - post:
          url: "/v1/chat"
          headers:
            Authorization: "Bearer {{ apiKey }}"
            Content-Type: "application/json"
          json:
            user_id: "load_test_user_{{ $randomNumber() }}"
            session_id: "session_{{ $randomNumber() }}"
            message: "Load test message {{ $randomString() }}"
            provider: "{{ $pick(['openai', 'anthropic', 'auto']) }}"
          expect:
            - statusCode: 200
            - hasHeader: "content-type"
          capture:
            - json: "$.response"
              as: "chatResponse"
            - json: "$.tokens_used"
              as: "tokensUsed"

  - name: "Health Check"
    weight: 10
    flow:
      - get:
          url: "/v1/health"
          expect:
            - statusCode: 200
            - hasProperty: "status"

  - name: "Analytics API"
    weight: 10
    flow:
      - get:
          url: "/v1/analytics/overview"
          headers:
            Authorization: "Bearer {{ apiKey }}"
          expect:
            - statusCode: 200
```

```javascript
// load-tests/k6-stress-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up
    { duration: '5m', target: 100 }, // Steady state
    { duration: '2m', target: 200 }, // Stress
    { duration: '5m', target: 200 }, // Stress steady
    { duration: '2m', target: 500 }, // Spike
    { duration: '2m', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.05'],   // Error rate under 5%
    errors: ['rate<0.05'],
  },
};

export default function() {
  const payload = JSON.stringify({
    user_id: `stress_user_${Math.floor(Math.random() * 1000)}`,
    session_id: `session_${Math.floor(Math.random() * 100)}`,
    message: `Stress test message ${Math.random()}`,
    provider: 'auto'
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${API_KEY}`,
    },
  };

  // Chat API request
  const chatResponse = http.post(`${BASE_URL}/v1/chat`, payload, params);
  
  const chatOk = check(chatResponse, {
    'chat status is 200': (r) => r.status === 200,
    'chat response time < 1000ms': (r) => r.timings.duration < 1000,
    'chat has response content': (r) => r.json('response') !== undefined,
  });
  
  if (!chatOk) {
    errorRate.add(1);
  }

  // Health check
  const healthResponse = http.get(`${BASE_URL}/v1/health`);
  check(healthResponse, {
    'health status is 200': (r) => r.status === 200,
    'health response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(1);
}

export function handleSummary(data) {
  return {
    'stress-test-results.json': JSON.stringify(data, null, 2),
    'stress-test-report.html': htmlReport(data),
  };
}
```

```go
// backend/cmd/profile/main.go
package main

import (
    "log"
    "os"
    "runtime/pprof"
    "time"
    
    "github.com/qt1-middleware/backend/middleware"
)

func main() {
    // CPU profiling
    cpuProfile, err := os.Create("cpu.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer cpuProfile.Close()
    
    if err := pprof.StartCPUProfile(cpuProfile); err != nil {
        log.Fatal(err)
    }
    defer pprof.StopCPUProfile()
    
    // Memory profiling
    defer func() {
        memProfile, err := os.Create("mem.prof")
        if err != nil {
            log.Fatal(err)
        }
        defer memProfile.Close()
        
        if err := pprof.WriteHeapProfile(memProfile); err != nil {
            log.Fatal(err)
        }
    }()
    
    // Run performance test workload
    proxy := middleware.NewProxy(/* config */)
    
    // Simulate load for profiling
    for i := 0; i < 10000; i++ {
        // Simulate chat requests
        request := middleware.ChatRequest{
            UserID:    fmt.Sprintf("profile_user_%d", i%100),
            SessionID: fmt.Sprintf("session_%d", i%10),
            Message:   fmt.Sprintf("Profile test message %d", i),
            Provider:  "openai",
        }
        
        // Process request
        _, err := proxy.ProcessChat(context.Background(), &request)
        if err != nil {
            log.Printf("Error processing request %d: %v", i, err)
        }
        
        if i%1000 == 0 {
            log.Printf("Processed %d requests", i)
        }
    }
    
    log.Println("Profiling complete. Run:")
    log.Println("go tool pprof cpu.prof")
    log.Println("go tool pprof mem.prof")
}
```

### Testing Requirements
- [ ] Test backend performance under various loads
- [ ] Test database query optimization
- [ ] Test memory usage and potential leaks
- [ ] Test concurrent request handling
- [ ] Test system limits and breaking points

### Acceptance Criteria
- [ ] API handles 1000+ concurrent requests
- [ ] 95th percentile response time <500ms
- [ ] Memory usage remains stable under load
- [ ] Database queries optimized for performance
- [ ] System degrades gracefully under extreme load
- [ ] No memory leaks detected in extended runs

### Risk Mitigation
- Use realistic test data and scenarios
- Monitor system resources during testing
- Implement gradual load increases

---

## **Cycle 19E: Coverage Analysis and Continuous Testing**
**Duration:** 3-4 hours | **Priority:** Low

### Prerequisites
- All previous cycles completed
- Understanding of coverage analysis tools
- Knowledge of CI/CD integration patterns

### Implementation Tasks
- [ ] Set up comprehensive coverage reporting
- [ ] Create coverage quality gates
- [ ] Implement mutation testing
- [ ] Add continuous testing in CI/CD
- [ ] Create test result dashboards
- [ ] Add test performance monitoring

### Code Deliverables
```bash
#!/bin/bash
# scripts/test-coverage.sh
set -e

echo "🧪 Running comprehensive test coverage analysis..."

# Backend coverage
echo "📊 Generating backend coverage..."
cd backend
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out

# Calculate coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Backend coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE < 90" | bc -l) )); then
    echo "❌ Backend coverage below 90% threshold"
    exit 1
fi

# Frontend coverage
echo "📊 Generating frontend coverage..."
cd ../frontend
npm run test:coverage

# Check coverage thresholds
npx nyc check-coverage --lines 85 --functions 85 --branches 80 --statements 85

# E2E coverage (if available)
echo "📊 Generating E2E coverage..."
cd ../e2e
npx playwright test --reporter=html

# Generate combined coverage report
echo "📈 Generating combined coverage report..."
cd ..
node scripts/combine-coverage.js

echo "✅ Coverage analysis complete!"
```

```javascript
// scripts/combine-coverage.js
const fs = require('fs');
const path = require('path');

const backendCoverage = parseCoverage('backend/coverage.out');
const frontendCoverage = require('./frontend/coverage/coverage-summary.json');

const combinedReport = {
  timestamp: new Date().toISOString(),
  backend: {
    lines: backendCoverage.lines,
    functions: backendCoverage.functions,
    statements: backendCoverage.statements
  },
  frontend: {
    lines: frontendCoverage.total.lines.pct,
    functions: frontendCoverage.total.functions.pct,
    branches: frontendCoverage.total.branches.pct,
    statements: frontendCoverage.total.statements.pct
  },
  overall: calculateOverallCoverage(backendCoverage, frontendCoverage)
};

// Write combined report
fs.writeFileSync(
  'coverage/combined-report.json', 
  JSON.stringify(combinedReport, null, 2)
);

console.log('Combined Coverage Report:');
console.log(`Backend: ${combinedReport.backend.lines}%`);
console.log(`Frontend: ${combinedReport.frontend.lines}%`);
console.log(`Overall: ${combinedReport.overall}%`);

function parseCoverage(filePath) {
  // Parse Go coverage output
  const content = fs.readFileSync(filePath, 'utf8');
  // Implementation would parse the coverage file
  return { lines: 92.5, functions: 89.3, statements: 91.7 };
}

function calculateOverallCoverage(backend, frontend) {
  // Weighted average based on lines of code
  const backendWeight = 0.6;
  const frontendWeight = 0.4;
  
  return (backend.lines * backendWeight + frontend.lines * frontendWeight).toFixed(1);
}
```

```yaml
# .github/workflows/test-coverage.yml
name: Test Coverage Analysis

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test-coverage:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_USER: testuser
          POSTGRES_DB: qt1_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '18'
        cache: 'npm'
        cache-dependency-path: frontend/package-lock.json
    
    - name: Install dependencies
      run: |
        cd backend && go mod download
        cd ../frontend && npm ci
        cd ../e2e && npm ci
    
    - name: Run backend tests with coverage
      run: |
        cd backend
        go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
        go tool cover -func=coverage.out
      env:
        DB_HOST: localhost
        DB_PORT: 5432
        REDIS_HOST: localhost
        REDIS_PORT: 6379
    
    - name: Run frontend tests with coverage
      run: |
        cd frontend
        npm run test:coverage
    
    - name: Install Playwright browsers
      run: |
        cd e2e
        npx playwright install --with-deps
    
    - name: Run E2E tests
      run: |
        cd e2e
        npx playwright test
      env:
        BASE_URL: http://localhost:3000
    
    - name: Generate combined coverage report
      run: |
        chmod +x scripts/test-coverage.sh
        ./scripts/test-coverage.sh
    
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        files: ./backend/coverage.out,./frontend/coverage/lcov.info
        flags: backend,frontend
        name: qt1-middleware-coverage
    
    - name: Upload test results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: test-results
        path: |
          backend/coverage.html
          frontend/coverage/
          e2e/playwright-report/
          coverage/combined-report.json
    
    - name: Comment coverage on PR
      if: github.event_name == 'pull_request'
      uses: marocchino/sticky-pull-request-comment@v2
      with:
        header: coverage
        path: coverage/coverage-comment.md

  mutation-testing:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install go-mutesting
      run: go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest
    
    - name: Run mutation testing
      run: |
        cd backend
        go-mutesting --target ./middleware --timeout 30s > mutation-results.txt
        cat mutation-results.txt
    
    - name: Upload mutation testing results
      uses: actions/upload-artifact@v3
      with:
        name: mutation-testing-results
        path: backend/mutation-results.txt
```

```javascript
// frontend/jest.config.js
module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/src/test/setup.ts'],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/test/**/*',
    '!src/**/*.stories.{ts,tsx}',
    '!src/main.tsx',
    '!src/vite-env.d.ts'
  ],
  coverageThreshold: {
    global: {
      lines: 85,
      functions: 85,
      branches: 80,
      statements: 85
    },
    './src/components/': {
      lines: 90,
      functions: 90,
      branches: 85,
      statements: 90
    },
    './src/hooks/': {
      lines: 95,
      functions: 95,
      branches: 90,
      statements: 95
    }
  },
  coverageReporters: ['text', 'lcov', 'html', 'json-summary'],
  testMatch: [
    '<rootDir>/src/**/__tests__/**/*.{ts,tsx}',
    '<rootDir>/src/**/*.{test,spec}.{ts,tsx}'
  ],
  transform: {
    '^.+\\.(ts|tsx)$': ['ts-jest', {
      tsconfig: 'tsconfig.json'
    }]
  },
  moduleNameMapping: {
    '^@/(.*)$': '<rootDir>/src/$1'
  }
};
```

### Testing Requirements
- [ ] Test coverage reports generate correctly
- [ ] Test quality gates enforce standards
- [ ] Test mutation testing identifies weak tests
- [ ] Test CI integration works reliably
- [ ] Test performance monitoring tracks trends

### Acceptance Criteria
- [ ] Combined test coverage >90% for critical paths
- [ ] Coverage reports accessible in CI/CD dashboard
- [ ] Quality gates prevent merging of low-coverage code
- [ ] Mutation testing score >80%
- [ ] Test execution time trends monitored
- [ ] Flaky test detection and reporting works

### Risk Mitigation
- Set realistic coverage thresholds
- Monitor test execution performance
- Implement proper test result retention

---

## **Integration Testing Checklist**
After all cycles complete:
- [ ] Full test suite integration (unit + integration + E2E)
- [ ] Cross-platform testing verification
- [ ] Performance testing under realistic load
- [ ] Coverage analysis across all test types
- [ ] CI/CD pipeline integration validation

## **Success Metrics**
- Overall test coverage >90% for critical components
- Test suite execution time <10 minutes in CI
- E2E test reliability >99% (non-flaky)
- Performance tests validate SLA requirements
- Zero critical bugs reach production
- Test maintenance overhead <20% of development time