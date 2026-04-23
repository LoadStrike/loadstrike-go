package loadstrike

import (
	"runtime"
)

const runtimeServiceBaseURL = "https://licensing.loadstrike.com"

func runtimeGOOS() string {
	return runtime.GOOS
}

func runtimeGOARCH() string {
	return runtime.GOARCH
}
