package loadstrike

import (
	"strconv"
	"sync"
)

type runtimeRegisteredCallback struct {
	ID           string
	ScenarioName string
	StepName     string
	Run          func(*stepRuntimeContext) replyResult
}

type runtimeCallbackRegistry struct {
	mu        sync.RWMutex
	callbacks map[string]runtimeRegisteredCallback
	nextID    int
}

func newRuntimeCallbackRegistry() *runtimeCallbackRegistry {
	return &runtimeCallbackRegistry{
		callbacks: map[string]runtimeRegisteredCallback{},
	}
}

func (r *runtimeCallbackRegistry) registerScenarioStep(scenarioName, stepName string, run func(*stepRuntimeContext) replyResult) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	id := "step-" + strconv.Itoa(r.nextID)
	r.callbacks[id] = runtimeRegisteredCallback{
		ID:           id,
		ScenarioName: scenarioName,
		StepName:     stepName,
		Run:          run,
	}
	return id
}

func (r *runtimeCallbackRegistry) lookup(id string) (runtimeRegisteredCallback, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.callbacks[id]
	return value, ok
}
