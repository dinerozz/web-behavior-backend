package entity

type DetailedAnalysis struct {
	DomainBreakdown   DomainBreakdown   `json:"domain_breakdown"`
	ProductivityScore ProductivityScore `json:"productivity_score"`
	BehaviorInsights  []string          `json:"behavior_insights"`
	KeyFindings       []string          `json:"key_findings"`
}

type DomainBreakdown struct {
	WorkTools     []string `json:"work_tools"`    // GitHub, Jira, etc.
	Development   []string `json:"development"`   // localhost, CodeSandbox, etc.
	Research      []string `json:"research"`      // Stack Overflow, docs, etc.
	Communication []string `json:"communication"` // Gmail, Telegram, etc.
	Distractions  []string `json:"distractions"`  // YouTube, social, etc.
}

type ProductivityScore struct {
	Overall     int    `json:"overall"`     // 0-100
	Focus       int    `json:"focus"`       // на основе deep work
	Efficiency  int    `json:"efficiency"`  // engagement rate
	Balance     int    `json:"balance"`     // work/life domains
	Explanation string `json:"explanation"` // краткое объяснение счета
}

type DomainAnalysis struct {
	FocusLevel      string           `json:"focus_level"`
	FocusInsight    string           `json:"focus_insight"`
	Recommendations []string         `json:"recommendations"`
	WorkPattern     string           `json:"work_pattern"`
	Analysis        DetailedAnalysis `json:"analysis"`
}
