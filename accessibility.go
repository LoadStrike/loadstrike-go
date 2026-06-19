package loadstrike

import (
	"fmt"
	"net/url"
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
	URL        string
	Violations []LoadStrikeAccessibilityViolation
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
