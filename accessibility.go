package loadstrike

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	accessibilityTestingFeature = "testing.accessibility"
	browserWebVitalsFeature     = "testing.browser_web_vitals"
)

type LoadStrikeAccessibilityOptions struct {
	URL                   string
	Standard              string
	MaxViolations         int
	MaxCriticalViolations int
	MaxSeriousViolations  int
	Tags                  []string
	IncludeSelectors      []string
	ExcludeSelectors      []string
}

type LoadStrikeAccessibilityViolation struct {
	RuleID      string
	Impact      string
	Description string
	HelpURL     string
	Targets     []string
}

type LoadStrikeAccessibilityResult struct {
	URL           string
	Violations    []LoadStrikeAccessibilityViolation
	KeyboardFocus LoadStrikeAccessibilityKeyboardFocusResult
}

type LoadStrikeAccessibilityKeyboardFocusResult struct {
	TabbableElementCount int
	FocusOrderWarnings   []string
	MissingFocusStyles   []string
}

func (r LoadStrikeAccessibilityResult) TotalViolations() int {
	return len(r.Violations)
}

func (r LoadStrikeAccessibilityResult) CriticalViolations() int {
	return r.countImpact("critical")
}

func (r LoadStrikeAccessibilityResult) SeriousViolations() int {
	return r.countImpact("serious")
}

func (r LoadStrikeAccessibilityResult) ModerateViolations() int {
	return r.countImpact("moderate")
}

func (r LoadStrikeAccessibilityResult) MinorViolations() int {
	return r.countImpact("minor")
}

func (r LoadStrikeAccessibilityResult) countImpact(impact string) int {
	count := 0
	for _, violation := range r.Violations {
		if strings.EqualFold(violation.Impact, impact) {
			count++
		}
	}
	return count
}

type LoadStrikeAccessibilityCheckContext struct {
	Options         LoadStrikeAccessibilityOptions
	ScenarioContext LoadStrikeScenarioContext
}

type loadStrikeAccessibilityNamespace struct{}

var LoadStrikeAccessibility loadStrikeAccessibilityNamespace

type LoadStrikeAccessibilityBaseline struct {
	AllowedRuleIDs []string
}

type LoadStrikeAccessibilityBaselineOutcome struct {
	Passed     bool
	NewRuleIDs []string
}

type LoadStrikeAccessibilityCiGate struct {
	MaxViolations         int
	MaxCriticalViolations int
	MaxSeriousViolations  int
	Baseline              LoadStrikeAccessibilityBaseline
}

type LoadStrikeAccessibilityCiGateOutcome struct {
	Passed  bool
	Reasons []string
}

func (loadStrikeAccessibilityNamespace) ScanHTML(options LoadStrikeAccessibilityOptions, html string) LoadStrikeAccessibilityResult {
	lower := strings.ToLower(html)
	violations := make([]LoadStrikeAccessibilityViolation, 0)
	add := func(ruleID, impact, description string) {
		violations = append(violations, LoadStrikeAccessibilityViolation{RuleID: ruleID, Impact: impact, Description: description})
	}
	if !regexp.MustCompile(`<html\b[^>]*\blang\s*=\s*["'][^"']+["']`).MatchString(lower) {
		add("html-lang", "serious", "The page should declare a language.")
	}
	if regexp.MustCompile(`<title>\s*</title>`).MatchString(lower) || !strings.Contains(lower, "<title") {
		add("document-title", "serious", "The page should have a useful title.")
	}
	imgTags := regexp.MustCompile(`<img\b[^>]*>`).FindAllString(lower, -1)
	for _, tag := range imgTags {
		if !strings.Contains(tag, " alt=") && !strings.Contains(tag, " alt\t=") {
			add("image-alt", "critical", "Images should provide alternate text.")
			break
		}
	}
	if regexp.MustCompile(`<button\b[^>]*>\s*</button>`).MatchString(lower) {
		add("button-name", "critical", "Buttons should have accessible names.")
	}
	inputTags := regexp.MustCompile(`<input\b[^>]*>`).FindAllString(lower, -1)
	for _, tag := range inputTags {
		if strings.Contains(tag, "type=\"hidden\"") || strings.Contains(tag, "type='hidden'") {
			continue
		}
		if strings.Contains(tag, "aria-label") || strings.Contains(tag, "aria-labelledby") || strings.Contains(lower, "<label") {
			continue
		}
		add("label", "critical", "Inputs should have labels.")
		break
	}
	keyboard := LoadStrikeAccessibilityKeyboardFocusResult{
		TabbableElementCount: len(regexp.MustCompile(`<(a|button|input|select|textarea)\b`).FindAllStringIndex(lower, -1)),
	}
	if !strings.Contains(lower, ":focus") && !strings.Contains(lower, "focus-visible") {
		keyboard.MissingFocusStyles = append(keyboard.MissingFocusStyles, "No visible focus style was detected.")
	}
	return LoadStrikeAccessibilityResult{URL: options.URL, Violations: violations, KeyboardFocus: keyboard}
}

func (b LoadStrikeAccessibilityBaseline) Compare(result LoadStrikeAccessibilityResult) LoadStrikeAccessibilityBaselineOutcome {
	allowed := map[string]struct{}{}
	for _, ruleID := range b.AllowedRuleIDs {
		if strings.TrimSpace(ruleID) != "" {
			allowed[strings.TrimSpace(ruleID)] = struct{}{}
		}
	}
	seen := map[string]struct{}{}
	newRuleIDs := make([]string, 0)
	for _, violation := range result.Violations {
		ruleID := strings.TrimSpace(violation.RuleID)
		if ruleID == "" {
			continue
		}
		if _, ok := allowed[ruleID]; ok {
			continue
		}
		if _, ok := seen[ruleID]; ok {
			continue
		}
		seen[ruleID] = struct{}{}
		newRuleIDs = append(newRuleIDs, ruleID)
	}
	return LoadStrikeAccessibilityBaselineOutcome{Passed: len(newRuleIDs) == 0, NewRuleIDs: newRuleIDs}
}

func (g LoadStrikeAccessibilityCiGate) Evaluate(result LoadStrikeAccessibilityResult) LoadStrikeAccessibilityCiGateOutcome {
	reasons := make([]string, 0)
	if g.MaxViolations >= 0 && result.TotalViolations() > g.MaxViolations {
		reasons = append(reasons, fmt.Sprintf("accessibility violations %d exceeded limit %d", result.TotalViolations(), g.MaxViolations))
	}
	if result.CriticalViolations() > g.MaxCriticalViolations {
		reasons = append(reasons, fmt.Sprintf("critical accessibility violations %d exceeded limit %d", result.CriticalViolations(), g.MaxCriticalViolations))
	}
	if result.SeriousViolations() > g.MaxSeriousViolations {
		reasons = append(reasons, fmt.Sprintf("serious accessibility violations %d exceeded limit %d", result.SeriousViolations(), g.MaxSeriousViolations))
	}
	if baseline := g.Baseline.Compare(result); !baseline.Passed {
		reasons = append(reasons, "new accessibility rule violations: "+strings.Join(baseline.NewRuleIDs, ", "))
	}
	return LoadStrikeAccessibilityCiGateOutcome{Passed: len(reasons) == 0, Reasons: reasons}
}

func (loadStrikeAccessibilityNamespace) CreateScenario(
	name string,
	options LoadStrikeAccessibilityOptions,
	check func(LoadStrikeAccessibilityCheckContext) (LoadStrikeAccessibilityResult, error),
) LoadStrikeScenario {
	if strings.TrimSpace(name) == "" {
		panic("Scenario name must be provided.")
	}
	validateAbsoluteURL(options.URL, "Accessibility check URL")
	if check == nil {
		panic("Accessibility check callback must be provided.")
	}

	native := createScenario(name, func(context LoadStrikeScenarioContext) LoadStrikeReply {
		result, err := check(LoadStrikeAccessibilityCheckContext{
			Options:         options,
			ScenarioContext: context,
		})
		if err != nil {
			return LoadStrikeResponse.Fail("accessibility", err.Error())
		}
		if failure := evaluateAccessibilityResult(options, result); failure != "" {
			return LoadStrikeResponse.Fail("accessibility", failure)
		}
		return LoadStrikeResponse.Ok(
			"200",
			int64(0),
			fmt.Sprintf("Accessibility check passed with %d violation(s).", result.TotalViolations()),
		)
	}).WithoutWarmUp().withInternalLicenseFeatures(accessibilityTestingFeature)

	return newLoadStrikeScenario(native)
}

type LoadStrikeBrowserWebVitalsOptions struct {
	URL                         string
	MaxLargestContentfulPaintMS *float64
	MaxInteractionToNextPaintMS *float64
	MaxCumulativeLayoutShift    *float64
	MaxFirstContentfulPaintMS   *float64
	MaxTimeToFirstByteMS        *float64
}

type LoadStrikeBrowserWebVitalsResult struct {
	URL                      string
	LargestContentfulPaintMS *float64
	InteractionToNextPaintMS *float64
	CumulativeLayoutShift    *float64
	FirstContentfulPaintMS   *float64
	TimeToFirstByteMS        *float64
}

type LoadStrikeBrowserWebVitalsContext struct {
	Options         LoadStrikeBrowserWebVitalsOptions
	ScenarioContext LoadStrikeScenarioContext
}

type loadStrikeBrowserWebVitalsNamespace struct{}

var LoadStrikeBrowserWebVitals loadStrikeBrowserWebVitalsNamespace

const webVitalsBrowserScript = `
(() => {
  const navigation = performance.getEntriesByType("navigation")[0];
  const paint = performance.getEntriesByType("paint");
  const fcp = paint.find((entry) => entry.name === "first-contentful-paint");
  return {
    largestContentfulPaintMs: globalThis.__loadstrikeLcp ?? globalThis.__loadstrike_lcp_ms ?? 0,
    interactionToNextPaintMs: globalThis.__loadstrikeInp ?? globalThis.__loadstrike_inp_ms ?? 0,
    cumulativeLayoutShift: globalThis.__loadstrikeCls ?? globalThis.__loadstrike_cls ?? 0,
    firstContentfulPaintMs: fcp ? fcp.startTime : 0,
    timeToFirstByteMs: navigation ? navigation.responseStart : 0
  };
})()
`

func (loadStrikeBrowserWebVitalsNamespace) FromPerformanceSnapshot(options LoadStrikeBrowserWebVitalsOptions, snapshot map[string]float64) LoadStrikeBrowserWebVitalsResult {
	return webVitalsResultFromSnapshot(options, snapshot)
}

func (loadStrikeBrowserWebVitalsNamespace) FromPlaywrightPageAsync(options LoadStrikeBrowserWebVitalsOptions, page any) (LoadStrikeBrowserWebVitalsResult, error) {
	validateAbsoluteURL(options.URL, "Browser Web Vitals URL")
	snapshot, err := invokeBrowserWebVitalsMethod(page, []string{"EvaluateAsync", "Evaluate", "evaluate"}, webVitalsBrowserScript)
	if err != nil {
		return LoadStrikeBrowserWebVitalsResult{}, err
	}
	return webVitalsResultFromSnapshot(options, snapshot), nil
}

func (loadStrikeBrowserWebVitalsNamespace) FromSeleniumDriver(options LoadStrikeBrowserWebVitalsOptions, driver any) (LoadStrikeBrowserWebVitalsResult, error) {
	validateAbsoluteURL(options.URL, "Browser Web Vitals URL")
	snapshot, err := invokeBrowserWebVitalsMethod(driver, []string{"ExecuteScript", "executeScript"}, webVitalsBrowserScript)
	if err != nil {
		return LoadStrikeBrowserWebVitalsResult{}, err
	}
	return webVitalsResultFromSnapshot(options, snapshot), nil
}

func webVitalsResultFromSnapshot(options LoadStrikeBrowserWebVitalsOptions, snapshot any) LoadStrikeBrowserWebVitalsResult {
	values := browserSnapshotValues(snapshot)
	value := func(keys ...string) *float64 {
		for _, key := range keys {
			if current, ok := values[key]; ok {
				copied := current
				return &copied
			}
		}
		return nil
	}
	return LoadStrikeBrowserWebVitalsResult{
		URL:                      options.URL,
		LargestContentfulPaintMS: value("LCP", "largestContentfulPaintMs"),
		InteractionToNextPaintMS: value("INP", "interactionToNextPaintMs"),
		CumulativeLayoutShift:    value("CLS", "cumulativeLayoutShift"),
		FirstContentfulPaintMS:   value("FCP", "firstContentfulPaintMs"),
		TimeToFirstByteMS:        value("TTFB", "timeToFirstByteMs"),
	}
}

func invokeBrowserWebVitalsMethod(target any, names []string, script string) (result any, err error) {
	if target == nil {
		return nil, fmt.Errorf("browser object must be provided")
	}
	value := reflect.ValueOf(target)
	for _, name := range names {
		method := value.MethodByName(name)
		if !method.IsValid() {
			continue
		}
		methodType := method.Type()
		if methodType.NumIn() > 0 && methodType.In(0).Kind() != reflect.String {
			continue
		}
		if !methodType.IsVariadic() && methodType.NumIn() > 1 {
			continue
		}

		var args []reflect.Value
		if methodType.NumIn() > 0 {
			args = append(args, reflect.ValueOf(script))
		}

		callResults, callErr := callBrowserWebVitalsMethod(method, args)
		if callErr != nil {
			return nil, callErr
		}
		if len(callResults) == 0 {
			return nil, nil
		}
		if last := callResults[len(callResults)-1]; last.IsValid() && last.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !last.IsNil() {
				return nil, last.Interface().(error)
			}
			if len(callResults) == 1 {
				return nil, nil
			}
		}
		return callResults[0].Interface(), nil
	}
	return nil, fmt.Errorf("browser object does not expose a supported Web Vitals evaluation method")
}

func callBrowserWebVitalsMethod(method reflect.Value, args []reflect.Value) (results []reflect.Value, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("browser Web Vitals evaluation failed: %v", recovered)
		}
	}()
	return method.Call(args), nil
}

func browserSnapshotValues(snapshot any) map[string]float64 {
	values := map[string]float64{}
	if snapshot == nil {
		return values
	}
	current := reflect.ValueOf(snapshot)
	for current.Kind() == reflect.Pointer || current.Kind() == reflect.Interface {
		if current.IsNil() {
			return values
		}
		current = current.Elem()
	}
	switch current.Kind() {
	case reflect.Map:
		for _, key := range current.MapKeys() {
			keyText := fmt.Sprint(key.Interface())
			if parsed, ok := browserSnapshotFloat(current.MapIndex(key).Interface()); ok {
				values[keyText] = parsed
			}
		}
	case reflect.Struct:
		currentType := current.Type()
		for i := 0; i < current.NumField(); i++ {
			field := currentType.Field(i)
			if field.PkgPath != "" {
				continue
			}
			if parsed, ok := browserSnapshotFloat(current.Field(i).Interface()); ok {
				values[field.Name] = parsed
			}
		}
	}
	return values
}

func browserSnapshotFloat(value any) (float64, bool) {
	if value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		reflected := reflect.ValueOf(value)
		if reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
			if reflected.IsNil() {
				return 0, false
			}
			return browserSnapshotFloat(reflected.Elem().Interface())
		}
		return 0, false
	}
}

func (loadStrikeBrowserWebVitalsNamespace) CreateScenario(
	name string,
	options LoadStrikeBrowserWebVitalsOptions,
	measure func(LoadStrikeBrowserWebVitalsContext) (LoadStrikeBrowserWebVitalsResult, error),
) LoadStrikeScenario {
	if strings.TrimSpace(name) == "" {
		panic("Scenario name must be provided.")
	}
	validateAbsoluteURL(options.URL, "Browser Web Vitals URL")
	if measure == nil {
		panic("Browser Web Vitals measurement callback must be provided.")
	}

	native := createScenario(name, func(context LoadStrikeScenarioContext) LoadStrikeReply {
		result, err := measure(LoadStrikeBrowserWebVitalsContext{
			Options:         options,
			ScenarioContext: context,
		})
		if err != nil {
			return LoadStrikeResponse.Fail("web_vitals", err.Error())
		}
		if failure := evaluateWebVitalsResult(options, result); failure != "" {
			return LoadStrikeResponse.Fail("web_vitals", failure)
		}
		return LoadStrikeResponse.Ok("200", int64(0), "Browser Web Vitals check passed.")
	}).WithoutWarmUp().withInternalLicenseFeatures(browserWebVitalsFeature)

	return newLoadStrikeScenario(native)
}

func validateAbsoluteURL(value string, label string) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" {
		panic(label + " must be an absolute URL.")
	}
}

func evaluateAccessibilityResult(
	options LoadStrikeAccessibilityOptions,
	result LoadStrikeAccessibilityResult,
) string {
	maxViolations := maxAccessibilityInt(options.MaxViolations, 0)
	maxCritical := maxAccessibilityInt(options.MaxCriticalViolations, 0)
	maxSerious := maxAccessibilityInt(options.MaxSeriousViolations, 0)
	if result.TotalViolations() > maxViolations {
		return fmt.Sprintf("Accessibility violations %d exceeded limit %d.", result.TotalViolations(), maxViolations)
	}
	if result.CriticalViolations() > maxCritical {
		return fmt.Sprintf("Critical accessibility violations %d exceeded limit %d.", result.CriticalViolations(), maxCritical)
	}
	if result.SeriousViolations() > maxSerious {
		return fmt.Sprintf("Serious accessibility violations %d exceeded limit %d.", result.SeriousViolations(), maxSerious)
	}
	return ""
}

func evaluateWebVitalsResult(
	options LoadStrikeBrowserWebVitalsOptions,
	result LoadStrikeBrowserWebVitalsResult,
) string {
	return firstWebVitalsViolation(
		"LCP", result.LargestContentfulPaintMS, options.MaxLargestContentfulPaintMS, "ms",
		"INP", result.InteractionToNextPaintMS, options.MaxInteractionToNextPaintMS, "ms",
		"CLS", result.CumulativeLayoutShift, options.MaxCumulativeLayoutShift, "",
		"FCP", result.FirstContentfulPaintMS, options.MaxFirstContentfulPaintMS, "ms",
		"TTFB", result.TimeToFirstByteMS, options.MaxTimeToFirstByteMS, "ms",
	)
}

func firstWebVitalsViolation(values ...any) string {
	for i := 0; i+3 < len(values); i += 4 {
		name, _ := values[i].(string)
		actual, _ := values[i+1].(*float64)
		limit, _ := values[i+2].(*float64)
		unit, _ := values[i+3].(string)
		if actual == nil || limit == nil || *actual <= *limit {
			continue
		}
		suffix := ""
		if strings.TrimSpace(unit) != "" {
			suffix = " " + unit
		}
		return fmt.Sprintf("%s %g%s exceeded limit %g%s.", name, *actual, suffix, *limit, suffix)
	}
	return ""
}

func maxAccessibilityInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
