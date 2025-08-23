
package diagnostics

type Severity string
const (
    Info Severity = "info"
    Warn Severity = "warning"
    Err  Severity = "error"
)

type Diagnostic struct {
    Severity       Severity       `json:"severity"`
    Code           string         `json:"code"`
    Summary        string         `json:"summary"`
    Detail         string         `json:"detail,omitempty"`
    LikelyCauses   []string       `json:"likely_causes,omitempty"`
    SuggestedFixes []string       `json:"suggested_fixes,omitempty"`
    Evidence       map[string]any `json:"evidence,omitempty"`
}
