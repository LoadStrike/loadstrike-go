package loadstrike

type loadStrikeAutopilotNamespace struct{}

var LoadStrikeAutopilot loadStrikeAutopilotNamespace

func (loadStrikeAutopilotNamespace) Generate(request LoadStrikeAutopilotRequest) LoadStrikeAutopilotResult {
	result, err := GenerateAutopilot(request)
	if err != nil {
		panic(err)
	}
	return result
}

func (loadStrikeAutopilotNamespace) generate(request LoadStrikeAutopilotRequest) LoadStrikeAutopilotResult {
	return LoadStrikeAutopilot.Generate(request)
}

func GenerateAutopilot(request LoadStrikeAutopilotRequest) (LoadStrikeAutopilotResult, error) {
	return runAutopilotViaPrivateRuntime(request)
}
