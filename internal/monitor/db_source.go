package monitor

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	logTypeConsume = 2
	logTypeError   = 5
)

// prompt 检查 / 泄漏保护拦截的请求也会写入 type=5 错误日志（other 带专属标记），
// 属于网关策略拦截而非模型调用失败，统计时整体排除，保持与 New API 性能面板口径一致。
// 占位符依次对应 logTypeConsume、logTypeError。
func logScopeCond() string {
	return `(type = ? OR (type = ?` +
		` AND COALESCE(other, '') NOT LIKE '%"prompt_check"%'` +
		` AND COALESCE(other, '') NOT LIKE '%"leak_protection_reason"%'))`
}

type DBSource struct {
	server *Server
	db     *sql.DB
	driver string
}

type dbConfig struct {
	driver string
	dsn    string
}

type tokenRow struct {
	id             int
	userID         int
	key            string
	status         int
	name           string
	createdTime    int64
	accessedTime   int64
	expiredTime    int64
	remainQuota    int64
	unlimitedQuota bool
	usedQuota      int64
	group          string
}

type userRow struct {
	id           int
	username     string
	displayName  string
	requestCount int
	remainQuota  int64
	usedQuota    int64
}

type channelRow struct {
	id           int
	channelType  int
	status       int
	autoBan      int
	tag          string
	responseTime int
	usedQuota    int64
	balance      float64
	priority     int
	weight       int
	createdTime  int64
	testTime     int64
}

func NewDBSource(server *Server) (*DBSource, error) {
	cfg, ok := readDBConfig()
	if !ok {
		return nil, fmt.Errorf("未配置 New API 数据库")
	}
	db, err := sql.Open(cfg.driver, cfg.dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 8))
	db.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 4))
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &DBSource{server: server, db: db, driver: cfg.driver}, nil
}

func (d *DBSource) Dashboard(label string, hours int) (DashboardData, error) {
	start := d.startTime(hours)
	filter := "created_at >= ? AND " + logScopeCond()
	args := []any{start, logTypeConsume, logTypeError}
	stats, err := d.hourlyStats(filter, args)
	if err != nil {
		return DashboardData{}, err
	}
	overview, err := d.overview(filter, args)
	if err != nil {
		return DashboardData{}, err
	}
	topModels, err := d.topModels(filter, args, 20)
	if err != nil {
		return DashboardData{}, err
	}
	return DashboardData{
		Hours:       label,
		Overview:    overview,
		HourlyStats: stats,
		TopModels:   topModels,
	}, nil
}

func (d *DBSource) Models(hours int) ([]map[string]string, error) {
	start := d.startTime(hours)
	query := d.rebind(`
		SELECT model_name
		FROM logs
		WHERE created_at >= ? AND ` + logScopeCond() + ` AND model_name <> ''
		GROUP BY model_name
		ORDER BY COUNT(*) DESC
		LIMIT 100
	`)
	rows, err := d.db.Query(query, start, logTypeConsume, logTypeError)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, map[string]string{"name": name})
	}
	return items, rows.Err()
}

func (d *DBSource) ModelLogs(name string, hours int) (map[string]any, error) {
	start := d.startTime(hours)
	filter := "created_at >= ? AND " + logScopeCond() + " AND model_name = ?"
	args := []any{start, logTypeConsume, logTypeError, name}
	stats, err := d.hourlyStats(filter, args)
	if err != nil {
		return nil, err
	}
	overview, err := d.overview(filter, args)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"summary": ModelSummary{
			ModelName:      name,
			TotalRecords:   overview.TotalRecords,
			SuccessRecords: overview.SuccessRecords,
			FailedRecords:  overview.FailedRecords,
			SuccessRate:    overview.SuccessRate,
			AvgTime:        overview.AvgTime,
			TotalQuota:     overview.TotalQuota,
			TotalTokens:    overview.TotalTokens,
			ActiveHours:    overview.ActiveHours,
			PeakCount:      peak(stats),
			FirstUsedAt:    overview.FirstSeenAt,
			LastUsedAt:     overview.LastSeenAt,
		},
		"hourly_stats": stats,
	}, nil
}

func (d *DBSource) KeyQuota(key string, hours int) (KeyData, error) {
	token, err := d.findToken(key)
	if err != nil {
		return KeyData{}, err
	}
	user, err := d.findUser(token.userID)
	if err != nil {
		return KeyData{}, err
	}
	start := d.startTime(hours)
	filter := "created_at >= ? AND " + logScopeCond() + " AND token_id = ?"
	args := []any{start, logTypeConsume, logTypeError, token.id}
	stats, err := d.hourlyStats(filter, args)
	if err != nil {
		return KeyData{}, err
	}
	overview, err := d.overview(filter, args)
	if err != nil {
		return KeyData{}, err
	}
	topModels, err := d.topModels(filter, args, 20)
	if err != nil {
		return KeyData{}, err
	}
	return KeyData{
		Token: TokenInfo{
			Name:           token.name,
			MaskedKey:      maskKey(token.key),
			Group:          token.group,
			Status:         token.status,
			UnlimitedQuota: token.unlimitedQuota,
			UsedQuota:      token.usedQuota,
			RemainQuota:    token.remainQuota,
			CreatedTime:    token.createdTime,
			AccessedTime:   token.accessedTime,
			ExpiredTime:    token.expiredTime,
		},
		User: UserInfo{
			Username:     user.username,
			DisplayName:  user.displayName,
			UserID:       user.id,
			RequestCount: user.requestCount,
			RemainQuota:  user.remainQuota,
			UsedQuota:    user.usedQuota,
		},
		UsageSummary: makeUsageSummary(overview, len(topModels)),
		HourlyStats:  stats,
		TopModels:    topModels,
	}, nil
}

func (d *DBSource) ChannelRecords(id int, hours int) (ChannelData, error) {
	channel, err := d.findChannel(id)
	if err != nil {
		return ChannelData{}, err
	}
	start := d.startTime(hours)
	filter := "created_at >= ? AND " + logScopeCond() + " AND channel_id = ?"
	args := []any{start, logTypeConsume, logTypeError, id}
	stats, err := d.hourlyStats(filter, args)
	if err != nil {
		return ChannelData{}, err
	}
	overview, err := d.overview(filter, args)
	if err != nil {
		return ChannelData{}, err
	}
	topModels, err := d.topModels(filter, args, 20)
	if err != nil {
		return ChannelData{}, err
	}
	return ChannelData{
		Channel: ChannelInfo{
			ChannelID:    channel.id,
			ChannelType:  channel.channelType,
			Status:       channel.status,
			AutoBan:      channel.autoBan,
			Tag:          channel.tag,
			ResponseTime: channel.responseTime,
			UsedQuota:    channel.usedQuota,
			Balance:      int(channel.balance),
			Priority:     channel.priority,
			Weight:       channel.weight,
			CreatedTime:  channel.createdTime,
			TestTime:     channel.testTime,
		},
		UsageSummary: makeUsageSummary(overview, len(topModels)),
		HourlyStats:  stats,
		TopModels:    topModels,
	}, nil
}

func (d *DBSource) hourlyStats(filter string, args []any) ([]HourlyStat, error) {
	query := d.rebind(fmt.Sprintf(`
		SELECT %s AS hour,
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS success,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(AVG(CASE WHEN type = %d AND use_time > 0 THEN use_time ELSE NULL END), 0) AS avg_time,
			COALESCE(SUM(CASE WHEN type = %d THEN quota ELSE 0 END), 0) AS total_quota
		FROM logs
		WHERE %s
		GROUP BY hour
		ORDER BY hour
	`, d.hourBucketExpr(), logTypeConsume, logTypeError, logTypeConsume, logTypeConsume, filter))
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make([]HourlyStat, 0)
	for rows.Next() {
		var item HourlyStat
		if err := rows.Scan(&item.Hour, &item.Total, &item.Success, &item.Failed, &item.AvgTime, &item.TotalQuota); err != nil {
			return nil, err
		}
		item.AvgTime = round2(item.AvgTime)
		stats = append(stats, item)
	}
	return stats, rows.Err()
}

func (d *DBSource) overview(filter string, args []any) (Overview, error) {
	query := d.rebind(fmt.Sprintf(`
		SELECT COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS success,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS failed,
			COALESCE(AVG(CASE WHEN type = %d AND use_time > 0 THEN use_time ELSE NULL END), 0) AS avg_time,
			COALESCE(SUM(CASE WHEN type = %d THEN quota ELSE 0 END), 0) AS total_quota,
			COALESCE(SUM(CASE WHEN type = %d THEN prompt_tokens + completion_tokens ELSE 0 END), 0) AS total_tokens,
			COALESCE(COUNT(DISTINCT %s), 0) AS active_hours,
			COALESCE(MIN(created_at), 0) AS first_seen_at,
			COALESCE(MAX(created_at), 0) AS last_seen_at
		FROM logs
		WHERE %s
	`, logTypeConsume, logTypeError, logTypeConsume, logTypeConsume, logTypeConsume, d.hourBucketExpr(), filter))
	var overview Overview
	if err := d.db.QueryRow(query, args...).Scan(
		&overview.TotalRecords,
		&overview.SuccessRecords,
		&overview.FailedRecords,
		&overview.AvgTime,
		&overview.TotalQuota,
		&overview.TotalTokens,
		&overview.ActiveHours,
		&overview.FirstSeenAt,
		&overview.LastSeenAt,
	); err != nil {
		return Overview{}, err
	}
	overview.SuccessRate = round2(percent(overview.SuccessRecords, overview.TotalRecords))
	overview.AvgTime = round2(overview.AvgTime)
	return overview, nil
}

func (d *DBSource) topModels(filter string, args []any, limit int) ([]TopModel, error) {
	query := d.rebind(fmt.Sprintf(`
		SELECT model_name,
			COUNT(*) AS count,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS success_count,
			COALESCE(SUM(CASE WHEN type = %d THEN 1 ELSE 0 END), 0) AS failed_count,
			COALESCE(AVG(CASE WHEN type = %d AND use_time > 0 THEN use_time ELSE NULL END), 0) AS avg_time,
			COALESCE(SUM(CASE WHEN type = %d THEN quota ELSE 0 END), 0) AS total_quota,
			COALESCE(SUM(CASE WHEN type = %d THEN prompt_tokens ELSE 0 END), 0) AS prompt_tokens,
			COALESCE(SUM(CASE WHEN type = %d THEN completion_tokens ELSE 0 END), 0) AS completion_tokens,
			COALESCE(MIN(created_at), 0) AS first_used_at,
			COALESCE(MAX(created_at), 0) AS last_used_at,
			COALESCE(COUNT(DISTINCT %s), 0) AS active_hours
		FROM logs
		WHERE %s AND model_name <> ''
		GROUP BY model_name
		ORDER BY count DESC
		LIMIT %d
	`, logTypeConsume, logTypeError, logTypeConsume, logTypeConsume, logTypeConsume, logTypeConsume, d.hourBucketExpr(), filter, limit))
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	models := make([]TopModel, 0)
	for rows.Next() {
		var model TopModel
		if err := rows.Scan(
			&model.Name,
			&model.Count,
			&model.SuccessCount,
			&model.FailedCount,
			&model.AvgTime,
			&model.TotalQuota,
			&model.TotalPromptTokens,
			&model.TotalCompletionTokens,
			&model.FirstUsedAt,
			&model.LastUsedAt,
			&model.ActiveHours,
		); err != nil {
			return nil, err
		}
		model.TotalTokens = model.TotalPromptTokens + model.TotalCompletionTokens
		model.SuccessRate = round2(percent(model.SuccessCount, model.Count))
		model.AvgTime = round2(model.AvgTime)
		model.AvgQuotaPerRequest = safeDiv(model.TotalQuota, int64(model.Count))
		model.AvgTotalTokens = safeDiv(model.TotalTokens, int64(model.Count))
		models = append(models, model)
	}
	return models, rows.Err()
}

func (d *DBSource) findToken(key string) (tokenRow, error) {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(key, "sk-"), "SK-")
	query := d.rebind(`
		SELECT id, user_id, ` + d.ident("key") + `, status, name, created_time, accessed_time, expired_time,
			remain_quota, unlimited_quota, used_quota, ` + d.ident("group") + `
		FROM tokens
		WHERE deleted_at IS NULL AND (` + d.ident("key") + ` = ? OR ` + d.ident("key") + ` = ? OR name = ?)
		ORDER BY id DESC
		LIMIT 1
	`)
	var token tokenRow
	err := d.db.QueryRow(query, key, trimmed, key).Scan(
		&token.id,
		&token.userID,
		&token.key,
		&token.status,
		&token.name,
		&token.createdTime,
		&token.accessedTime,
		&token.expiredTime,
		&token.remainQuota,
		&token.unlimitedQuota,
		&token.usedQuota,
		&token.group,
	)
	if errorsIsNoRows(err) {
		return tokenRow{}, ErrNotFound
	}
	return token, err
}

func (d *DBSource) findUser(id int) (userRow, error) {
	query := d.rebind(`
		SELECT id, username, display_name, request_count, quota, used_quota
		FROM users
		WHERE id = ?
		LIMIT 1
	`)
	var user userRow
	err := d.db.QueryRow(query, id).Scan(
		&user.id,
		&user.username,
		&user.displayName,
		&user.requestCount,
		&user.remainQuota,
		&user.usedQuota,
	)
	if errorsIsNoRows(err) {
		return userRow{}, ErrNotFound
	}
	return user, err
}

func (d *DBSource) findChannel(id int) (channelRow, error) {
	query := d.rebind(`
		SELECT id, type, status, COALESCE(auto_ban, 0), COALESCE(tag, name), response_time,
			used_quota, balance, COALESCE(priority, 0), COALESCE(weight, 0), created_time, test_time
		FROM channels
		WHERE id = ?
		LIMIT 1
	`)
	var channel channelRow
	err := d.db.QueryRow(query, id).Scan(
		&channel.id,
		&channel.channelType,
		&channel.status,
		&channel.autoBan,
		&channel.tag,
		&channel.responseTime,
		&channel.usedQuota,
		&channel.balance,
		&channel.priority,
		&channel.weight,
		&channel.createdTime,
		&channel.testTime,
	)
	if errorsIsNoRows(err) {
		return channelRow{}, ErrNotFound
	}
	return channel, err
}

func (d *DBSource) startTime(hours int) int64 {
	return d.server.now().Add(-time.Duration(hours) * time.Hour).Unix()
}

func (d *DBSource) hourBucketExpr() string {
	return "created_at - MOD(created_at, 3600)"
}

func (d *DBSource) ident(name string) string {
	if d.driver == "postgres" {
		return `"` + name + `"`
	}
	return "`" + name + "`"
}

func (d *DBSource) rebind(query string) string {
	if d.driver != "postgres" {
		return query
	}
	var builder strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			builder.WriteString("$")
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func readDBConfig() (dbConfig, bool) {
	dsn := firstEnv("NEW_API_DSN", "DATABASE_URL", "SQL_DSN", "MYSQL_URL", "POSTGRES_URL")
	if dsn == "" {
		return dbConfig{}, false
	}
	driver := strings.ToLower(firstEnv("NEW_API_DB_DRIVER", "DATABASE_DRIVER"))
	if driver == "" {
		driver = detectDriver(dsn)
	}
	if driver == "postgresql" {
		driver = "postgres"
	}
	if driver == "mysql" && strings.HasPrefix(strings.ToLower(dsn), "mysql://") {
		if normalized, err := normalizeMySQLURL(dsn); err == nil {
			dsn = normalized
		}
	}
	return dbConfig{driver: driver, dsn: dsn}, true
}

func detectDriver(dsn string) string {
	lower := strings.ToLower(dsn)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return "postgres"
	}
	return "mysql"
}

func normalizeMySQLURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	password, _ := parsed.User.Password()
	user := parsed.User.Username()
	params := parsed.Query()
	if params.Get("parseTime") == "" {
		params.Set("parseTime", "true")
	}
	if params.Get("charset") == "" {
		params.Set("charset", "utf8mb4")
	}
	return fmt.Sprintf("%s:%s@tcp(%s)%s?%s", user, password, parsed.Host, parsed.Path, params.Encode()), nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}
