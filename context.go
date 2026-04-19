package loadstrike

type contextState struct {
	RunnerKey      string
	ReportsEnabled bool
	scenarios      []scenarioDefinition
}

// LoadStrikeContext is the reusable configured execution context.
type LoadStrikeContext struct {
	native *contextState
}

func wrapLoadStrikeContext(context *contextState) LoadStrikeContext {
	return LoadStrikeContext{native: context}
}

func (c LoadStrikeContext) nativeValue() *contextState {
	return c.native
}

func requireNativeContext(context *contextState) *contextState {
	if context == nil {
		panic("context must be provided")
	}
	return context
}
