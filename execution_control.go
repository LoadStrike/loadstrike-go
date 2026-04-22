package loadstrike

import (
	stdcontext "context"
	"strings"
	"sync"
)

type executionControl struct {
	testCtx          stdcontext.Context
	testCancel       stdcontext.CancelFunc
	mu               sync.Mutex
	testStopped      bool
	stoppedScenarios map[string]string
}

func newExecutionControl() *executionControl {
	ctx, cancel := stdcontext.WithCancel(stdcontext.Background())
	return &executionControl{
		testCtx:          ctx,
		testCancel:       cancel,
		stoppedScenarios: map[string]string{},
	}
}

func (c *executionControl) scenarioContext(scenarioName string) (stdcontext.Context, stdcontext.CancelFunc) {
	if c == nil {
		return stdcontext.WithCancel(stdcontext.Background())
	}
	return stdcontext.WithCancel(c.testCtx)
}

func (c *executionControl) stopTest(reason string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	alreadyStopped := c.testStopped
	c.testStopped = true
	c.mu.Unlock()
	if !alreadyStopped && c.testCancel != nil {
		c.testCancel()
	}
}

func (c *executionControl) stopScenario(scenarioName string, reason string) {
	if c == nil {
		return
	}
	name := strings.TrimSpace(scenarioName)
	if name == "" {
		return
	}
	c.mu.Lock()
	c.stoppedScenarios[strings.ToLower(name)] = reason
	c.mu.Unlock()
}

func (c *executionControl) shouldStopTest() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.testStopped
}

func (c *executionControl) shouldStopScenario(scenarioName string) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.stoppedScenarios[strings.ToLower(strings.TrimSpace(scenarioName))]
	return ok
}
