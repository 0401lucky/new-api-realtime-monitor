# 监控台前端项目 - API 接口文档

> 本文档供后端开发使用，前端代码在 `index.html` 中。

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Tailwind CSS | 3.4.17 (CDN) | 样式框架 |
| ECharts | 5.5.1 (CDN) | 图表（折线图、柱状图、雷达图） |
| Lucide Icons | 0.468.0 (CDN) | SVG 图标库 |
| Google Fonts | Inter + JetBrains Mono | 字体 |

## API 接口列表

所有接口均需携带 cookie（`credentials: 'include'`）进行身份认证。

### 1. GET /api/config — 站点配置

**响应示例：**
```json
{
  "data": {
    "cacheTtlSeconds": 900,
    "docsLink": "https://docs.newapi.pro",
    "geetestCaptchaId": "",
    "logo": "https://example.com/logo.jpg",
    "quotaPerUnit": 500000,
    "searchVerificationEnabled": false,
    "serverAddress": "https://chybenzun.top",
    "startTime": 1783052911,
    "systemName": "CHY公益站",
    "version": "v0.0.0"
  }
}
```

### 2. GET /api/dashboard?hours={hours} — 仪表盘总览

**参数：** `hours` = 1|2|6|12|24|48|72|168|all

**响应示例：**
```json
{
  "data": {
    "hours": "24",
    "overview": {
      "totalRecords": 12345,
      "successRecords": 12000,
      "failedRecords": 345,
      "successRate": 97.20,
      "avgTime": 1.23,
      "totalQuota": 5000000,
      "totalTokens": 12345678,
      "activeHours": 24,
      "firstSeenAt": 1782960000,
      "lastSeenAt": 1783046400
    },
    "hourly_stats": [
      {
        "hour": 1782960000,
        "total": 500,
        "success": 490,
        "failed": 10,
        "avgTime": 1.1
      }
    ],
    "top_models": [
      {
        "name": "gpt-4",
        "count": 5000,
        "successRate": 98.5,
        "avgTime": 2.3,
        "totalTokens": 5000000,
        "totalQuota": 2000000,
        "totalPromptTokens": 3000000,
        "totalCompletionTokens": 2000000,
        "failedCount": 75,
        "lastUsedAt": 1783046400,
        "activeHours": 24
      }
    ]
  }
}
```

### 3. GET /api/logs/models?hours={hours} — 模型列表

**响应示例：**
```json
{
  "data": [
    { "name": "gpt-4" },
    { "name": "gpt-3.5-turbo" },
    { "name": "claude-3-opus" }
  ]
}
```

### 4. GET /api/logs?model_name={name}&hours={hours} — 单个模型趋势

**响应示例：**
```json
{
  "data": {
    "summary": {
      "modelName": "gpt-4",
      "totalRecords": 2000,
      "successRecords": 1960,
      "failedRecords": 40,
      "successRate": 98.0,
      "avgTime": 2.1,
      "totalQuota": 1000000,
      "totalTokens": 3000000,
      "activeHours": 20,
      "peakCount": 150,
      "firstUsedAt": 1782960000,
      "lastUsedAt": 1783046400
    },
    "hourly_stats": [
      {
        "hour": 1782960000,
        "total": 100,
        "success": 98,
        "failed": 2,
        "avgTime": 2.0
      }
    ]
  }
}
```

### 5. GET /api/key/quota?key={apiKey}&hours={hours} — 查询 Key 详情

**参数：** `key` = API Key（以 `sk-` 开头）或任意 key 名称

**响应示例：**
```json
{
  "data": {
    "token": {
      "name": "我的Key",
      "maskedKey": "sk-****abcd",
      "group": "default",
      "status": 1,
      "unlimitedQuota": false,
      "usedQuota": 1500000,
      "remainQuota": 3500000,
      "createdTime": 1782000000,
      "accessedTime": 1783046400,
      "expiredTime": -1
    },
    "user": {
      "username": "admin",
      "displayName": "管理员",
      "userId": 1,
      "requestCount": 50000,
      "remainQuota": 5000000,
      "usedQuota": 20000000
    },
    "usage_summary": {
      "totalRecords": 3000,
      "successRecords": 2950,
      "failedRecords": 50,
      "successRate": 98.33,
      "avgTime": 1.5,
      "totalQuota": 1500000,
      "totalTokens": 5000000,
      "totalPromptTokens": 3000000,
      "totalCompletionTokens": 2000000,
      "modelCount": 5
    },
    "hourly_stats": [
      {
        "hour": 1782960000,
        "total": 120,
        "success": 118,
        "failed": 2
      }
    ],
    "top_models": [
      {
        "name": "gpt-4",
        "count": 1000,
        "successRate": 99.0,
        "avgTime": 2.1,
        "totalTokens": 2000000,
        "totalQuota": 500000,
        "totalPromptTokens": 1200000,
        "totalCompletionTokens": 800000,
        "successCount": 990,
        "failedCount": 10,
        "firstUsedAt": 1782960000,
        "lastUsedAt": 1783046400,
        "avgQuotaPerRequest": 500,
        "avgTotalTokens": 2000
      }
    ]
  }
}
```

### 6. GET /api/channel/records?channel_id={id}&hours={hours} — 查询渠道详情

**参数：** `channel_id` = 纯数字渠道 ID

**响应示例：**
```json
{
  "data": {
    "channel": {
      "channelId": 1,
      "channelType": 1,
      "status": 1,
      "autoBan": 0,
      "tag": "GPT渠道",
      "responseTime": 1200,
      "usedQuota": 500000,
      "balance": 100,
      "priority": 1,
      "weight": 10,
      "createdTime": 1782000000,
      "testTime": 1783046400
    },
    "usage_summary": {
      "totalRecords": 5000,
      "successRecords": 4800,
      "failedRecords": 200,
      "successRate": 96.0,
      "avgTime": 2.0,
      "totalQuota": 1000000,
      "totalTokens": 8000000,
      "totalPromptTokens": 5000000,
      "totalCompletionTokens": 3000000
    },
    "hourly_stats": [
      {
        "hour": 1782960000,
        "total": 200,
        "success": 192,
        "failed": 8,
        "totalQuota": 40000
      }
    ],
    "top_models": [
      {
        "name": "gpt-4",
        "count": 2000,
        "successRate": 97.5,
        "avgTime": 2.2,
        "totalTokens": 4000000,
        "totalQuota": 500000,
        "totalPromptTokens": 2500000,
        "totalCompletionTokens": 1500000,
        "successCount": 1950,
        "failedCount": 50,
        "firstUsedAt": 1782960000,
        "lastUsedAt": 1783046400,
        "avgQuotaPerRequest": 250,
        "avgTotalTokens": 2000
      }
    ]
  }
}
```

## 关键字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `quotaPerUnit` | number | 每单位 Quota 对应的额度（前端用于显示换算），默认 500000 |
| `status` | number | 1=正常，其他=已禁用 |
| `unlimitedQuota` | bool | 是否无限额度 |
| `expiredTime` | number | -1 表示永不过期 |
| `hour` | number | Unix 时间戳（秒） |
| `autoBan` | number | 1=开启自动封禁，0=关闭 |

## 前端搜索逻辑

搜索框输入判断规则：
- 以 `sk-` 开头 → 调用 `/api/key/quota`
- 纯数字 → 调用 `/api/channel/records`
- 其他 → 默认调用 `/api/key/quota`

## 前端状态管理

- 主题模式存储：`localStorage.theme-mode`（`light` | `dark` | `auto`）
- 用户信息存储：`localStorage.user`（JSON，含 `id` 字段）
- 自动刷新间隔：`cacheTtlSeconds * 2/3` 秒
