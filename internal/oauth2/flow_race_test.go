package oauth2

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestManagerConcurrentFlows starts, waits on and cancels flows from several
// goroutines at once, as the app's bindings and callback goroutine do. Run
// with -race to catch unguarded access to the session and callback server.
func TestManagerConcurrentFlows(t *testing.T) {
	m := NewManager()
	provider := &ProviderConfig{Name: "test", ClientID: "id", AuthURL: "https://auth.example/authorize", TokenURL: "https://auth.example/token"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := m.StartAuthFlowWithProvider(ctx, provider); err != nil {
					t.Error(err)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				m.CancelAuthFlow()
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Returns once its server is stopped by a cancel or restart.
				waitCtx, waitCancel := context.WithTimeout(ctx, 50*time.Millisecond)
				_, _, _ = m.WaitForCallback(waitCtx)
				waitCancel()
			}
		}()
	}
	wg.Wait()
	m.CancelAuthFlow()
}
