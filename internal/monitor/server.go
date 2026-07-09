package monitor

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	staticDir  string
	config     Config
	now        func() time.Time
	source     DataSource
	authToken  string
	keyLimiter *rateLimiter
}

type Config struct {
	CacheTTLSeconds          int    `json:"cacheTtlSeconds"`
	DocsLink                 string `json:"docsLink"`
	GeetestCaptchaID         string `json:"geetestCaptchaId"`
	Logo                     string `json:"logo"`
	QuotaPerUnit             int64  `json:"quotaPerUnit"`
	SearchVerificationEnable bool   `json:"searchVerificationEnabled"`
	ServerAddress            string `json:"serverAddress"`
	StartTime                int64  `json:"startTime"`
	SystemName               string `json:"systemName"`
	Version                  string `json:"version"`
	AuthRequired             bool   `json:"authRequired"`
}

type HourlyStat struct {
	Hour       int64   `json:"hour"`
	Total      int     `json:"total"`
	Success    int     `json:"success"`
	Failed     int     `json:"failed"`
	AvgTime    float64 `json:"avgTime,omitempty"`
	TotalQuota int64   `json:"totalQuota,omitempty"`
}

type Overview struct {
	TotalRecords   int     `json:"totalRecords"`
	SuccessRecords int     `json:"successRecords"`
	FailedRecords  int     `json:"failedRecords"`
	SuccessRate    float64 `json:"successRate"`
	AvgTime        float64 `json:"avgTime"`
	TotalQuota     int64   `json:"totalQuota"`
	TotalTokens    int64   `json:"totalTokens"`
	ActiveHours    int     `json:"activeHours"`
	FirstSeenAt    int64   `json:"firstSeenAt"`
	LastSeenAt     int64   `json:"lastSeenAt"`
}

type TopModel struct {
	Name                  string  `json:"name"`
	Count                 int     `json:"count"`
	SuccessRate           float64 `json:"successRate"`
	AvgTime               float64 `json:"avgTime"`
	TotalTokens           int64   `json:"totalTokens"`
	TotalQuota            int64   `json:"totalQuota"`
	TotalPromptTokens     int64   `json:"totalPromptTokens"`
	TotalCompletionTokens int64   `json:"totalCompletionTokens"`
	SuccessCount          int     `json:"successCount"`
	FailedCount           int     `json:"failedCount"`
	FirstUsedAt           int64   `json:"firstUsedAt"`
	LastUsedAt            int64   `json:"lastUsedAt"`
	ActiveHours           int     `json:"activeHours,omitempty"`
	AvgQuotaPerRequest    int64   `json:"avgQuotaPerRequest,omitempty"`
	AvgTotalTokens        int64   `json:"avgTotalTokens,omitempty"`
}

type DashboardData struct {
	Hours       string       `json:"hours"`
	Overview    Overview     `json:"overview"`
	HourlyStats []HourlyStat `json:"hourly_stats"`
	TopModels   []TopModel   `json:"top_models"`
}

type ModelSummary struct {
	ModelName      string  `json:"modelName"`
	TotalRecords   int     `json:"totalRecords"`
	SuccessRecords int     `json:"successRecords"`
	FailedRecords  int     `json:"failedRecords"`
	SuccessRate    float64 `json:"successRate"`
	AvgTime        float64 `json:"avgTime"`
	TotalQuota     int64   `json:"totalQuota"`
	TotalTokens    int64   `json:"totalTokens"`
	ActiveHours    int     `json:"activeHours"`
	PeakCount      int     `json:"peakCount"`
	FirstUsedAt    int64   `json:"firstUsedAt"`
	LastUsedAt     int64   `json:"lastUsedAt"`
}

type UsageSummary struct {
	TotalRecords          int     `json:"totalRecords"`
	SuccessRecords        int     `json:"successRecords"`
	FailedRecords         int     `json:"failedRecords"`
	SuccessRate           float64 `json:"successRate"`
	AvgTime               float64 `json:"avgTime"`
	TotalQuota            int64   `json:"totalQuota"`
	TotalTokens           int64   `json:"totalTokens"`
	TotalPromptTokens     int64   `json:"totalPromptTokens"`
	TotalCompletionTokens int64   `json:"totalCompletionTokens"`
	ModelCount            int     `json:"modelCount,omitempty"`
}

type TokenInfo struct {
	Name           string `json:"name"`
	MaskedKey      string `json:"maskedKey"`
	Group          string `json:"group"`
	Status         int    `json:"status"`
	UnlimitedQuota bool   `json:"unlimitedQuota"`
	UsedQuota      int64  `json:"usedQuota"`
	RemainQuota    int64  `json:"remainQuota"`
	CreatedTime    int64  `json:"createdTime"`
	AccessedTime   int64  `json:"accessedTime"`
	ExpiredTime    int64  `json:"expiredTime"`
}

type UserInfo struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	UserID       int    `json:"userId"`
	RequestCount int    `json:"requestCount"`
	RemainQuota  int64  `json:"remainQuota"`
	UsedQuota    int64  `json:"usedQuota"`
}

type KeyData struct {
	Token        TokenInfo    `json:"token"`
	User         UserInfo     `json:"user"`
	UsageSummary UsageSummary `json:"usage_summary"`
	HourlyStats  []HourlyStat `json:"hourly_stats"`
	TopModels    []TopModel   `json:"top_models"`
}

type ChannelInfo struct {
	ChannelID    int    `json:"channelId"`
	ChannelType  int    `json:"channelType"`
	Status       int    `json:"status"`
	AutoBan      int    `json:"autoBan"`
	Tag          string `json:"tag"`
	ResponseTime int    `json:"responseTime"`
	UsedQuota    int64  `json:"usedQuota"`
	Balance      int    `json:"balance"`
	Priority     int    `json:"priority"`
	Weight       int    `json:"weight"`
	CreatedTime  int64  `json:"createdTime"`
	TestTime     int64  `json:"testTime"`
}

type ChannelData struct {
	Channel      ChannelInfo  `json:"channel"`
	UsageSummary UsageSummary `json:"usage_summary"`
	HourlyStats  []HourlyStat `json:"hourly_stats"`
	TopModels    []TopModel   `json:"top_models"`
}

var modelCatalog = []struct {
	name    string
	weight  float64
	latency float64
}{
	{"deepseek-chat", 1.00, 0.86},
	{"deepseek-reasoner", 0.78, 1.42},
	{"gpt-4o-mini", 0.70, 0.74},
	{"claude-3-5-sonnet", 0.54, 1.28},
	{"gemini-1.5-pro", 0.46, 1.11},
	{"qwen-plus", 0.38, 0.69},
	{"gpt-4o", 0.33, 1.04},
	{"glm-4-plus", 0.26, 0.82},
}

func New(staticDir string) *Server {
	cfg := loadConfig()
	authToken := strings.TrimSpace(os.Getenv("MONITOR_TOKEN"))
	cfg.AuthRequired = authToken != ""

	// 查询缓存：优先 QUERY_CACHE_TTL_SECONDS，否则跟随 CACHE_TTL_SECONDS，默认 60s
	queryTTL := envInt("QUERY_CACHE_TTL_SECONDS", 0)
	if queryTTL <= 0 {
		queryTTL = cfg.CacheTTLSeconds
	}
	if queryTTL <= 0 {
		queryTTL = 60
	}

	server := &Server{
		staticDir:  staticDir,
		config:     cfg,
		now:        time.Now,
		authToken:  authToken,
		keyLimiter: newRateLimiter(envInt("KEY_RATE_LIMIT", 30), time.Minute),
	}
	server.source = NewCachedSource(NewDataSource(server), queryTTL)
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/logs/models", s.handleModels)
	mux.HandleFunc("GET /api/logs", s.handleModelLogs)
	mux.HandleFunc("GET /api/key/quota", s.withKeyRateLimit(s.handleKeyQuota))
	mux.HandleFunc("GET /api/channel/records", s.withKeyRateLimit(s.handleChannelRecords))
	mux.HandleFunc("/", s.handleStatic)
	return withCommonHeaders(s.withAuth(mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"status": "ok", "time": s.now().Unix()},
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"data": s.config})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	label, hours, err := parseHours(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.source.Dashboard(label, hours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	_, hours, err := parseHours(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := s.source.Models(hours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) handleModelLogs(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("model_name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "model_name 不能为空")
		return
	}
	_, hours, err := parseHours(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.source.ModelLogs(name, hours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) handleKeyQuota(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key == "" {
		writeError(w, http.StatusBadRequest, "key 不能为空")
		return
	}
	_, hours, err := parseHours(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.source.KeyQuota(key, hours)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) handleChannelRecords(w http.ResponseWriter, r *http.Request) {
	rawID := strings.TrimSpace(r.URL.Query().Get("channel_id"))
	id, err := strconv.Atoi(rawID)
	if rawID == "" || err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "channel_id 必须是正整数")
		return
	}
	_, hours, err := parseHours(r.URL.Query().Get("hours"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.source.ChannelRecords(id, hours)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		http.ServeFile(w, r, filepath.Join(s.staticDir, path))
		return
	}
	http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
}

func (s *Server) makeSeries(hours int, offset int, scale float64) []HourlyStat {
	now := s.now().Truncate(time.Hour)
	start := now.Add(-time.Duration(hours-1) * time.Hour)
	stats := make([]HourlyStat, 0, hours)
	for i := 0; i < hours; i++ {
		wave := math.Sin(float64(i+offset)*0.55)*42 + math.Cos(float64(i+offset)*0.18)*28
		trend := float64(i) * 1.8
		base := 180 + wave + trend + float64((i+offset)%7)*9
		total := int(math.Max(8, math.Round(base*scale)))
		failRate := 0.018 + float64((i+offset)%6)*0.006
		failed := int(math.Round(float64(total) * failRate))
		if failed < 1 && total > 40 {
			failed = 1
		}
		success := total - failed
		avgTime := round2(0.62 + math.Abs(math.Sin(float64(i+offset)*0.21))*1.15 + float64(offset%4)*0.08)
		tokens := int64(total) * int64(1100+(i+offset)%900)
		stats = append(stats, HourlyStat{
			Hour:       start.Add(time.Duration(i) * time.Hour).Unix(),
			Total:      total,
			Success:    success,
			Failed:     failed,
			AvgTime:    avgTime,
			TotalQuota: tokens / 3,
		})
	}
	return stats
}

func (s *Server) makeTopModels(stats []HourlyStat, scale float64) []TopModel {
	overview := makeOverview(stats)
	result := make([]TopModel, 0, len(modelCatalog))
	for idx, model := range modelCatalog {
		count := int(math.Round(float64(overview.TotalRecords) * model.weight * scale / 3.2))
		if count < 1 {
			count = 1
		}
		failed := int(math.Round(float64(count) * (0.018 + float64(idx%5)*0.007)))
		success := count - failed
		totalTokens := int64(count) * int64(850+idx*210)
		totalQuota := totalTokens / int64(2+idx%3)
		result = append(result, TopModel{
			Name:                  model.name,
			Count:                 count,
			SuccessRate:           round2(percent(success, count)),
			AvgTime:               round2(model.latency + float64(idx%3)*0.09),
			TotalTokens:           totalTokens,
			TotalQuota:            totalQuota,
			TotalPromptTokens:     int64(float64(totalTokens) * 0.62),
			TotalCompletionTokens: int64(float64(totalTokens) * 0.38),
			SuccessCount:          success,
			FailedCount:           failed,
			FirstUsedAt:           overview.FirstSeenAt,
			LastUsedAt:            overview.LastSeenAt - int64(idx*300),
			ActiveHours:           overview.ActiveHours,
			AvgQuotaPerRequest:    safeDiv(totalQuota, int64(count)),
			AvgTotalTokens:        safeDiv(totalTokens, int64(count)),
		})
	}
	return result
}

func makeOverview(stats []HourlyStat) Overview {
	var total, success, failed int
	var quota int64
	var weightedTime float64
	for _, item := range stats {
		total += item.Total
		success += item.Success
		failed += item.Failed
		quota += item.TotalQuota
		weightedTime += item.AvgTime * float64(item.Total)
	}
	if len(stats) == 0 {
		return Overview{}
	}
	totalTokens := quota * 3
	avgTime := 0.0
	if total > 0 {
		avgTime = weightedTime / float64(total)
	}
	return Overview{
		TotalRecords:   total,
		SuccessRecords: success,
		FailedRecords:  failed,
		SuccessRate:    round2(percent(success, total)),
		AvgTime:        round2(avgTime),
		TotalQuota:     quota,
		TotalTokens:    totalTokens,
		ActiveHours:    len(stats),
		FirstSeenAt:    stats[0].Hour,
		LastSeenAt:     stats[len(stats)-1].Hour,
	}
}

func makeUsageSummary(overview Overview, modelCount int) UsageSummary {
	if modelCount == 0 {
		modelCount = len(modelCatalog)
	}
	return UsageSummary{
		TotalRecords:          overview.TotalRecords,
		SuccessRecords:        overview.SuccessRecords,
		FailedRecords:         overview.FailedRecords,
		SuccessRate:           overview.SuccessRate,
		AvgTime:               overview.AvgTime,
		TotalQuota:            overview.TotalQuota,
		TotalTokens:           overview.TotalTokens,
		TotalPromptTokens:     int64(float64(overview.TotalTokens) * 0.61),
		TotalCompletionTokens: int64(float64(overview.TotalTokens) * 0.39),
		ModelCount:            modelCount,
	}
}

func parseHours(value string) (string, int, error) {
	if strings.TrimSpace(value) == "" {
		return "24", 24, nil
	}
	if strings.EqualFold(value, "all") {
		return "all", 168, nil
	}
	allowed := map[int]bool{1: true, 2: true, 6: true, 12: true, 24: true, 48: true, 72: true, 168: true}
	hours, err := strconv.Atoi(value)
	if err != nil || !allowed[hours] {
		return "", 0, errors.New("hours 只支持 1、2、6、12、24、48、72、168 或 all")
	}
	return strconv.Itoa(hours), hours, nil
}

func loadConfig() Config {
	return Config{
		CacheTTLSeconds:          envInt("CACHE_TTL_SECONDS", 90),
		DocsLink:                 envString("DOCS_LINK", "https://docs.newapi.pro"),
		GeetestCaptchaID:         envString("GEETEST_CAPTCHA_ID", ""),
		Logo:                     envString("LOGO_URL", ""),
		QuotaPerUnit:             int64(envInt("QUOTA_PER_UNIT", 500000)),
		SearchVerificationEnable: envBool("SEARCH_VERIFICATION_ENABLED", false),
		ServerAddress:            envString("SERVER_ADDRESS", "/"),
		StartTime:                int64(envInt("START_TIME", int(time.Now().AddDate(0, -1, 0).Unix()))),
		SystemName:               envString("SYSTEM_NAME", "CHY公益站"),
		Version:                  envString("APP_VERSION", "v1.0.0"),
		AuthRequired:             false,
	}
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"message": message})
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func peak(stats []HourlyStat) int {
	maxValue := 0
	for _, item := range stats {
		if item.Total > maxValue {
			maxValue = item.Total
		}
	}
	return maxValue
}

func modelIndex(name string) int {
	for idx, model := range modelCatalog {
		if strings.EqualFold(model.name, name) {
			return idx
		}
	}
	return stableSeed(name) % len(modelCatalog)
}

func modelWeight(name string) float64 {
	idx := modelIndex(name)
	if idx >= 0 && idx < len(modelCatalog) {
		return math.Max(0.22, modelCatalog[idx].weight)
	}
	return 0.5
}

func stableSeed(value string) int {
	sum := 0
	for _, r := range value {
		sum += int(r)
	}
	if sum < 0 {
		return -sum
	}
	return sum
}

func tokenName(key string) string {
	if strings.HasPrefix(strings.ToLower(key), "sk-") {
		return "生产 API Key"
	}
	return key
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:3] + "****" + key[len(key)-4:]
}

func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) * 100 / float64(total)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func safeDiv(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	return a / b
}
