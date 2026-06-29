package i18n

var VI = map[string]string{
	"app.title":              "D Research CLI",
	"app.subtitle":           "Nghiên cứu sâu với phê duyệt plan bắt buộc",
	"nav.overview":           "Tổng quan",
	"nav.agents":             "Agent đang chạy",
	"nav.usage":              "Usage / Chi phí",
	"nav.cache":              "Cache",
	"view.stream":            "Dòng",
	"view.graph":             "Graph",
	"view.split":             "Chia đôi",
	"tab.plan":               "Plan",
	"tab.evidence":           "Evidence",
	"tab.gaps":               "Gaps",
	"tab.blockers":           "Blockers",
	"tab.artifacts":          "Artifacts",
	"tab.scenario":           "Scenario",
	"tab.outcomes":           "Outcomes",
	"tab.paths":              "Paths",
	"tab.sensitivity":        "Sensitivity",
	"settings.title":         "Cài đặt",
	"settings.models":        "Models",
	"settings.search":        "Search",
	"settings.browser":       "Browser",
	"settings.security":      "Security",
	"settings.runtime":       "Runtime",
	"cmd.palette":            "Command palette",
	"cmd.connect":            "Kết nối provider/model",
	"status.awaiting":        "Chờ phê duyệt plan",
	"status.running":         "Đang chạy",
	"status.blocked":         "Bị chặn",
	"status.done":            "Hoàn tất",
	"prob.uncalibrated":      "Uncalibrated",
}

func T(key string) string {
	if v, ok := VI[key]; ok {
		return v
	}
	return key
}