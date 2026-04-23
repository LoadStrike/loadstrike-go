package loadstrike

// LoadStrikeTask mirrors a void-returning async contract in a Go-friendly shape.
type LoadStrikeTask struct {
	await func() error
}

// Await executes the task and returns its completion error.
func (t LoadStrikeTask) Await() error {
	if t.await == nil {
		return nil
	}
	return t.await()
}

// CompletedTask returns a completed successful task.
func CompletedTask() LoadStrikeTask {
	return LoadStrikeTask{await: func() error { return nil }}
}

// TaskFromError returns a task that completes with the provided error.
func TaskFromError(err error) LoadStrikeTask {
	return LoadStrikeTask{await: func() error { return err }}
}

// TaskFromErrorString returns a task that completes with a string error.
func TaskFromErrorString(message string) LoadStrikeTask {
	return LoadStrikeTask{await: func() error {
		if message == "" {
			return nil
		}
		return simpleTaskError(message)
	}}
}

type simpleTaskError string

// Error returns the current error text. Use this when you need the readable failure message.
func (e simpleTaskError) Error() string { return string(e) }

// LoadStrikeValueTask mirrors a typed async contract in a Go-friendly shape.
type LoadStrikeValueTask[T any] struct {
	await func() (T, error)
}

// Await executes the task and returns its result and completion error.
func (t LoadStrikeValueTask[T]) Await() (T, error) {
	var zero T
	if t.await == nil {
		return zero, nil
	}
	return t.await()
}

// TaskFromResult returns a successful typed task.
func TaskFromResult[T any](value T) LoadStrikeValueTask[T] {
	return LoadStrikeValueTask[T]{await: func() (T, error) { return value, nil }}
}

// TaskFromResultError returns a typed task that completes with an error.
func TaskFromResultError[T any](err error) LoadStrikeValueTask[T] {
	return LoadStrikeValueTask[T]{await: func() (T, error) {
		var zero T
		return zero, err
	}}
}

// LoadStrikeBoolTask mirrors a boolean-returning async contract.
type LoadStrikeBoolTask = LoadStrikeValueTask[bool]

// TaskFromBool returns a successful boolean task.
func TaskFromBool(value bool) LoadStrikeBoolTask {
	return TaskFromResult(value)
}
