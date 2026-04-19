package loadstrike

import "strings"

type replyResult struct {
	IsSuccess  bool
	StatusCode string
	Message    string
}

// LoadStrikeReply is the public reply contract returned from scenario callbacks.
type LoadStrikeReply struct {
	native replyResult
}

func (r LoadStrikeReply) toNative() replyResult {
	return r.native
}

type loadStrikeResponseNamespace struct{}

// LoadStrikeResponse exposes reply builders matching the website samples.
var LoadStrikeResponse loadStrikeResponseNamespace

func (loadStrikeResponseNamespace) Ok(args ...any) LoadStrikeReply {
	statusCode := "200"
	message := ""
	if len(args) > 0 {
		if value, ok := args[0].(string); ok && strings.TrimSpace(value) != "" {
			statusCode = value
		}
	}
	if len(args) > 1 {
		if value, ok := args[1].(string); ok {
			message = value
		}
	}
	return LoadStrikeReply{
		native: replyResult{
			IsSuccess:  true,
			StatusCode: statusCode,
			Message:    message,
		},
	}
}
