package monitor

import (
	"errors"
	"strconv"
)

var ErrNotFound = errors.New("未找到对应记录")

type DataSource interface {
	Dashboard(label string, hours int) (DashboardData, error)
	Models(hours int) ([]map[string]string, error)
	ModelLogs(name string, hours int) (map[string]any, error)
	KeyQuota(key string, hours int) (KeyData, error)
	ChannelRecords(id int, hours int) (ChannelData, error)
}

type DemoSource struct {
	server *Server
}

func NewDataSource(server *Server) DataSource {
	if _, ok := readDBConfig(); ok {
		source, err := NewDBSource(server)
		if err != nil {
			return ErrorSource{err: err}
		}
		return source
	}
	return &DemoSource{server: server}
}

type ErrorSource struct {
	err error
}

func (e ErrorSource) Dashboard(label string, hours int) (DashboardData, error) {
	return DashboardData{}, e.err
}

func (e ErrorSource) Models(hours int) ([]map[string]string, error) {
	return nil, e.err
}

func (e ErrorSource) ModelLogs(name string, hours int) (map[string]any, error) {
	return nil, e.err
}

func (e ErrorSource) KeyQuota(key string, hours int) (KeyData, error) {
	return KeyData{}, e.err
}

func (e ErrorSource) ChannelRecords(id int, hours int) (ChannelData, error) {
	return ChannelData{}, e.err
}

func (d *DemoSource) Dashboard(label string, hours int) (DashboardData, error) {
	stats := d.server.makeSeries(hours, 0, 1)
	return DashboardData{
		Hours:       label,
		Overview:    makeOverview(stats),
		HourlyStats: stats,
		TopModels:   d.server.makeTopModels(stats, 1),
	}, nil
}

func (d *DemoSource) Models(hours int) ([]map[string]string, error) {
	items := make([]map[string]string, 0, len(modelCatalog))
	for _, model := range modelCatalog {
		items = append(items, map[string]string{"name": model.name})
	}
	return items, nil
}

func (d *DemoSource) ModelLogs(name string, hours int) (map[string]any, error) {
	idx := modelIndex(name)
	stats := d.server.makeSeries(hours, idx+1, modelWeight(name))
	overview := makeOverview(stats)
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

func (d *DemoSource) KeyQuota(key string, hours int) (KeyData, error) {
	seed := stableSeed(key)
	stats := d.server.makeSeries(hours, seed%9+1, 0.62+float64(seed%40)/100)
	overview := makeOverview(stats)
	topModels := d.server.makeTopModels(stats, 0.72)
	usedQuota := overview.TotalQuota + int64(seed%800000)
	remainQuota := int64(2600000 + seed%5000000)

	return KeyData{
		Token: TokenInfo{
			Name:           tokenName(key),
			MaskedKey:      maskKey(key),
			Group:          "default",
			Status:         1,
			UnlimitedQuota: false,
			UsedQuota:      usedQuota,
			RemainQuota:    remainQuota,
			CreatedTime:    d.server.now().AddDate(0, -2, -int(seed%14)).Unix(),
			AccessedTime:   overview.LastSeenAt,
			ExpiredTime:    -1,
		},
		User: UserInfo{
			Username:     "admin",
			DisplayName:  "管理员",
			UserID:       1,
			RequestCount: overview.TotalRecords + int(seed%9000),
			RemainQuota:  remainQuota * 2,
			UsedQuota:    usedQuota * 3,
		},
		UsageSummary: makeUsageSummary(overview, len(topModels)),
		HourlyStats:  stats,
		TopModels:    topModels,
	}, nil
}

func (d *DemoSource) ChannelRecords(id int, hours int) (ChannelData, error) {
	stats := d.server.makeSeries(hours, id%11+3, 0.76+float64(id%35)/100)
	overview := makeOverview(stats)
	topModels := d.server.makeTopModels(stats, 0.82)
	return ChannelData{
		Channel: ChannelInfo{
			ChannelID:    id,
			ChannelType:  id%4 + 1,
			Status:       1,
			AutoBan:      id % 2,
			Tag:          "生产渠道-" + strconv.Itoa(id),
			ResponseTime: int(round2(overview.AvgTime * 1000)),
			UsedQuota:    overview.TotalQuota,
			Balance:      100 + id%900,
			Priority:     id%10 + 1,
			Weight:       5 + id%20,
			CreatedTime:  d.server.now().AddDate(0, -3, -id%20).Unix(),
			TestTime:     overview.LastSeenAt,
		},
		UsageSummary: makeUsageSummary(overview, 0),
		HourlyStats:  stats,
		TopModels:    topModels,
	}, nil
}
