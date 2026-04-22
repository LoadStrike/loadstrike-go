package loadstrike

// NodeType identifies the execution role for a run.
type NodeType int

const (
	NodeTypeSingleNode NodeType = iota
	NodeTypeCoordinator
	NodeTypeAgent
)

const (
	LoadStrikeNodeTypeSingleNode  = NodeTypeSingleNode
	LoadStrikeNodeTypeCoordinator = NodeTypeCoordinator
	LoadStrikeNodeTypeAgent       = NodeTypeAgent
)

// LoadStrikeOperationType mirrors the .NET public operation enum.
type LoadStrikeOperationType int

const (
	LoadStrikeOperationTypeNone LoadStrikeOperationType = iota
	LoadStrikeOperationTypeInit
	LoadStrikeOperationTypeWarmUp
	LoadStrikeOperationTypeBombing
	LoadStrikeOperationTypeStop
	LoadStrikeOperationTypeComplete
	LoadStrikeOperationTypeError
)

// LoadStrikeScenarioOperation mirrors the .NET public scenario-operation enum.
type LoadStrikeScenarioOperation int

const (
	LoadStrikeScenarioOperationInit LoadStrikeScenarioOperation = iota
	LoadStrikeScenarioOperationClean
	LoadStrikeScenarioOperationWarmUp
	LoadStrikeScenarioOperationBombing
)

// RuntimePolicyErrorMode controls how runtime policy failures are surfaced.
type RuntimePolicyErrorMode int

const (
	RuntimePolicyErrorModeFail RuntimePolicyErrorMode = iota
	RuntimePolicyErrorModeContinue
)

const (
	LoadStrikeRuntimePolicyErrorModeFail     = RuntimePolicyErrorModeFail
	LoadStrikeRuntimePolicyErrorModeContinue = RuntimePolicyErrorModeContinue
)

const (
	LoadStrikeReportFormatHtml = ReportFormatHTML
	LoadStrikeReportFormatTxt  = ReportFormatTXT
	LoadStrikeReportFormatCsv  = ReportFormatCSV
	LoadStrikeReportFormatMd   = ReportFormatMD
)
