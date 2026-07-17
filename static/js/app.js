// ==================== 常量与配置 ====================
    const HOURS_PRESETS = [
      { value: '1', label: '1H' },
      { value: '2', label: '2H' },
      { value: '6', label: '6H' },
      { value: '12', label: '12H' },
      { value: '24', label: '1D' },
      { value: '48', label: '2D' },
      { value: '72', label: '3D' },
      { value: '168', label: '7D' },
      { value: 'all', label: 'ALL' },
    ];

    let quotaPerUnit = 500000;
    let autoRefreshTimer = null;

    // ==================== 全局状态 ====================
    const state = {
      loading: false,
      data: null,
      hours: '24',
      detailData: null,
      detailMode: null,
      detailExpandedModelName: null,
      topModelsView: 'table',
      expandedModelName: null,
      trendMode: 'all',
      trendModelName: '__all__',
      trendModelData: null,
      modelOptions: [],
    };

    // ==================== ECharts 实例引用 ====================
    let trendChartInstance = null;
    let topModelsChartInstance = null;
    let keyTrendChartInstance = null;
    let chartResizeBound = false;
    let chartResizeTimer = null;
    let scrollIdleTimer = null;

    function resizeAllCharts() {
      if (trendChartInstance) trendChartInstance.resize();
      if (topModelsChartInstance) topModelsChartInstance.resize();
      if (keyTrendChartInstance) keyTrendChartInstance.resize();
    }

    function ensureChartResizeListener() {
      if (chartResizeBound) return;
      chartResizeBound = true;
      window.addEventListener('resize', () => {
        if (chartResizeTimer) clearTimeout(chartResizeTimer);
        chartResizeTimer = setTimeout(resizeAllCharts, 120);
      }, { passive: true });
    }

    /** 滚动时暂停背景动画，停滚后恢复 */
    function bindScrollPerfHints() {
      const onScroll = () => {
        document.body.classList.add('is-scrolling');
        if (scrollIdleTimer) clearTimeout(scrollIdleTimer);
        scrollIdleTimer = setTimeout(() => {
          document.body.classList.remove('is-scrolling');
        }, 140);
      };
      window.addEventListener('scroll', onScroll, { passive: true });
      window.addEventListener('wheel', onScroll, { passive: true });
      window.addEventListener('touchmove', onScroll, { passive: true });
    }

    // ==================== DOM 元素引用 ====================
    const elements = {
      keySearchInput: document.getElementById('keySearchInput'),
      keySearchButton: document.getElementById('keySearchButton'),
      keyClearButton: document.getElementById('keyClearButton'),
      keyInfoSection: document.getElementById('keyInfoSection'),
      dashboardContent: document.getElementById('dashboardContent'),
      keyInfoCards: document.getElementById('keyInfoCards'),
      detailMetaGrid: document.getElementById('detailMetaGrid'),
      keyTrendChart: document.getElementById('keyTrendChart'),
      keyModelsTable: document.getElementById('keyModelsTable'),
      hoursButtonGroup: document.getElementById('hoursButtonGroup'),
      refreshButton: document.getElementById('refreshButton'),
      overviewCards: document.getElementById('overviewCards'),
      trendChart: document.getElementById('trendChart'),
      trendModelSelectButton: document.getElementById('trendModelSelectButton'),
      trendModelSelectLabel: document.getElementById('trendModelSelectLabel'),
      trendModelDropdown: document.getElementById('trendModelDropdown'),
      trendModelSearchInput: document.getElementById('trendModelSearchInput'),
      trendModelOptions: document.getElementById('trendModelOptions'),
      trendModelCards: document.getElementById('trendModelCards'),
      topModelsTable: document.getElementById('topModelsTable'),
      topModelsTableView: document.getElementById('topModelsTableView'),
      topModelsChartView: document.getElementById('topModelsChartView'),
      errorState: document.getElementById('errorState'),
      errorMessage: document.getElementById('errorMessage'),
    };

    // ==================== 工具函数 ====================
    function formatNumber(value) { return Number(value || 0).toLocaleString('zh-CN'); }
    function formatPercent(value) { return `${Number(value || 0).toFixed(2)}%`; }
    function formatSeconds(value) { return `${Number(value || 0).toFixed(2)}s`; }
    function formatCreditsFromQuota(value) {
      const credits = Number(value || 0) / quotaPerUnit;
      return credits.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    }
    function formatTimestamp(value) {
      if (!value) return '-';
      return new Date(Number(value) * 1000).toLocaleString('zh-CN', { hour12: false });
    }
    function formatNullableText(value, fallback = '-') {
      if (value === null || value === undefined) return fallback;
      const text = String(value).trim();
      return text ? text : fallback;
    }
    function formatStatusLabel(value) { return Number(value) === 1 ? '正常' : '已禁用'; }
    function formatTimeLabel(value, isMinuteLevel) {
      if (!value) return '-';
      const date = new Date(Number(value) * 1000);
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hour = String(date.getHours()).padStart(2, '0');
      const minute = String(date.getMinutes()).padStart(2, '0');
      if (isMinuteLevel) return `${hour}:${minute}`;
      return `${month}-${day} ${hour}:00`;
    }
    function detectMinuteLevel(hourlyStats) {
      if (hourlyStats.length < 2) return false;
      const gap = Math.abs(Number(hourlyStats[1].hour) - Number(hourlyStats[0].hour));
      return gap > 0 && gap <= 120;
    }
    function getMonitorToken() {
      return (localStorage.getItem('monitor-token') || '').trim();
    }

    function setMonitorToken(token) {
      const value = String(token || '').trim();
      if (value) localStorage.setItem('monitor-token', value);
      else localStorage.removeItem('monitor-token');
    }

    function getAuthHeaders() {
      const headers = {};
      const monitorToken = getMonitorToken();
      if (monitorToken) headers['X-Monitor-Token'] = monitorToken;
      try {
        const saved = window.localStorage.getItem('user');
        if (saved) {
          const user = JSON.parse(saved);
          if (user && user.id) headers['New-Api-User'] = String(user.id);
        }
      } catch (_) { /* ignore */ }
      return headers;
    }

    async function fetchAPI(url, options = {}) {
      const response = await fetch(url, {
        ...options,
        credentials: 'include',
        headers: { ...getAuthHeaders(), ...(options.headers || {}) },
      });
      if (response.status === 401) {
        const token = window.prompt('此监控台已开启访问保护，请输入访问令牌：', getMonitorToken());
        if (token !== null && token.trim()) {
          setMonitorToken(token.trim());
          return fetch(url, {
            ...options,
            credentials: 'include',
            headers: { ...getAuthHeaders(), ...(options.headers || {}) },
          });
        }
      }
      return response;
    }

    function successRateClass(rate) {
      const value = Number(rate || 0);
      if (value >= 99) return 'rate-good';
      if (value >= 95) return 'rate-warn';
      return 'rate-bad';
    }

    // ==================== 主题管理 ====================
    const systemThemeMedia = window.matchMedia('(prefers-color-scheme: dark)');

    function getThemeMode() {
      const mode = (localStorage.getItem('theme-mode') || '').trim().toLowerCase();
      if (mode === 'light' || mode === 'dark' || mode === 'auto') return mode;
      const legacy = (localStorage.getItem('newapi-monitor-theme') || '').trim().toLowerCase();
      if (legacy === 'light' || legacy === 'dark') return legacy;
      return 'auto';
    }

    function resolveTheme(mode = getThemeMode()) {
      if (mode === 'light' || mode === 'dark') return mode;
      return systemThemeMedia.matches ? 'dark' : 'light';
    }

    function applyTheme() {
      const mode = getThemeMode();
      const theme = resolveTheme(mode);
      document.documentElement.classList.toggle('dark', theme === 'dark');
      localStorage.setItem('theme-mode', mode);
      updateThemeIcon();
      // 更新图表主题
      updateChartsTheme();
    }

    function toggleTheme() {
      const currentMode = getThemeMode();
      const nextMode = currentMode === 'light' ? 'dark' : currentMode === 'dark' ? 'auto' : 'light';
      localStorage.setItem('theme-mode', nextMode);
      applyTheme();
      lucide.createIcons();
    }

    function updateThemeIcon() {
      const icon = document.getElementById('themeIcon');
      if (icon) {
        const isDark = document.documentElement.classList.contains('dark');
        icon.setAttribute('data-lucide', isDark ? 'moon' : 'sun');
      }
    }

    function updateChartsTheme() {
      // 主题切换时重绘图表
      if (state.data) {
        if (state.trendMode === 'all') {
          renderTrend(state.data.hourly_stats, null);
        } else if (state.trendModelData) {
          renderTrend(state.trendModelData.hourly_stats || [], state.trendModelData);
        }
        if (state.topModelsView === 'chart' && state.data.top_models?.length) {
          renderTopModelsChart(state.data.top_models);
        }
      }
      if (state.detailData) {
        renderKeyTrendChart(state.detailData.hourly_stats || []);
      }
    }

    // ==================== 错误/加载状态 ====================
    function setLoading(loading) {
      state.loading = loading;
      elements.refreshButton.disabled = loading;
      if (loading) {
        elements.refreshButton.innerHTML = '<i data-lucide="loader-circle" class="h-5 w-5 animate-spin"></i>';
      } else {
        elements.refreshButton.innerHTML = '<i data-lucide="refresh-cw" class="h-5 w-5"></i>';
      }
      lucide.createIcons();
    }

    function showError(message) {
      elements.errorMessage.textContent = message;
      elements.errorState.classList.remove('hidden');
    }

    function hideError() {
      elements.errorState.classList.add('hidden');
      elements.errorMessage.textContent = '';
    }

    // ==================== 时间范围分段控件 ====================
    function renderHoursButtons() {
      elements.hoursButtonGroup.innerHTML = HOURS_PRESETS.map(preset => {
        const isActive = state.hours === preset.value;
        return `<button type="button" class="${isActive ? 'is-active' : ''}" data-hours-btn="${preset.value}">${preset.label}</button>`;
      }).join('');

      elements.hoursButtonGroup.querySelectorAll('[data-hours-btn]').forEach(button => {
        button.addEventListener('click', () => {
          const value = button.getAttribute('data-hours-btn') || '24';
          if (state.hours === value) return;
          state.hours = value;
          state.expandedModelName = null;
          renderHoursButtons();
          hideError();
          if (state.detailData && elements.keySearchInput.value.trim()) {
            loadDetailBySearch(elements.keySearchInput.value);
          } else {
            loadDashboard();
          }
        });
      });
    }

    // ==================== 概览卡片 ====================
    function renderOverview(overview) {
      const cards = [
        { label: '总请求数', value: formatNumber(overview.totalRecords), hint: `成功 ${formatNumber(overview.successRecords)} / 失败 ${formatNumber(overview.failedRecords)}`, icon: 'waypoints', tone: 'teal' },
        { label: '成功率', value: formatPercent(overview.successRate), hint: `平均耗时 ${formatSeconds(overview.avgTime)}`, icon: 'badge-check', tone: 'emerald' },
        { label: '消耗额度', value: formatCreditsFromQuota(overview.totalQuota), hint: `原始 Quota ${formatNumber(overview.totalQuota)}`, icon: 'coins', tone: 'amber' },
        { label: '总 Tokens', value: formatNumber(overview.totalTokens), hint: `活跃小时 ${formatNumber(overview.activeHours)}`, icon: 'cpu', tone: 'sky' },
      ];

      elements.overviewCards.innerHTML = cards.map((card, index) => `
        <article class="glass glass-hover metric-card metric-card--${card.tone} rise-in rounded-[24px] p-6" style="animation-delay:${index * 40}ms">
          <div class="flex items-start justify-between gap-4">
            <div class="text-xs font-extrabold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">${card.label}</div>
            <span class="metric-icon shrink-0"><i data-lucide="${card.icon}" class="h-4 w-4"></i></span>
          </div>
          <div class="metric-value mt-4 text-3xl font-black text-slate-900 dark:text-white">${card.value}</div>
          <div class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">${card.hint}</div>
        </article>
      `).join('');
    }

    // ==================== 趋势图（ECharts） ====================
    function compactTrendStats(hourlyStats, maxPoints = 72) {
      if (!Array.isArray(hourlyStats) || hourlyStats.length <= maxPoints) return hourlyStats || [];

      const bucketSize = Math.ceil(hourlyStats.length / maxPoints);
      const compacted = [];

      for (let start = 0; start < hourlyStats.length; start += bucketSize) {
        const bucket = hourlyStats.slice(start, start + bucketSize);
        const total = bucket.reduce((sum, item) => sum + Number(item.total || 0), 0);
        const success = bucket.reduce((sum, item) => sum + Number(item.success || 0), 0);
        const failed = bucket.reduce((sum, item) => sum + Number(item.failed || 0), 0);
        const totalQuota = bucket.reduce((sum, item) => sum + Number(item.totalQuota || 0), 0);
        const weightedTime = bucket.reduce((sum, item) => sum + Number(item.avgTime || 0) * Math.max(1, Number(item.total || 0)), 0);
        const weight = bucket.reduce((sum, item) => sum + Math.max(1, Number(item.total || 0)), 0);

        compacted.push({
          hour: bucket[0].hour,
          total,
          success,
          failed,
          totalQuota,
          avgTime: weight ? Number((weightedTime / weight).toFixed(2)) : 0,
        });
      }

      return compacted;
    }

    function renderTrend(hourlyStats, context = null) {
      if (!hourlyStats.length) {
        if (trendChartInstance) trendChartInstance.clear();
        elements.trendChart.innerHTML = '<div class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">暂无趋势数据</div>';
        return;
      }

      const maxTotal = Math.max(...hourlyStats.map(item => item.total), 1);
      const trendSummaryText = document.getElementById('trendSummaryText');
      const trendRangeText = document.getElementById('trendRangeText');

      if (context?.summary) {
        trendSummaryText.textContent = `${context.summary.modelName} | 成功率 ${formatPercent(context.summary.successRate)} | 峰值 ${formatNumber(maxTotal)} 次请求`;
        trendRangeText.textContent = `模型时间范围：${formatTimestamp(context.summary.firstUsedAt)} - ${formatTimestamp(context.summary.lastUsedAt)}`;
      } else {
        trendSummaryText.textContent = `峰值 ${formatNumber(maxTotal)} 次请求`;
        if (state.data?.overview) {
          trendRangeText.textContent = `数据时间范围：${formatTimestamp(state.data.overview.firstSeenAt)} - ${formatTimestamp(state.data.overview.lastSeenAt)}`;
        }
      }

      if (!trendChartInstance) {
        trendChartInstance = echarts.init(elements.trendChart, null, { renderer: 'canvas' });
        ensureChartResizeListener();
      }

      const isDark = document.documentElement.classList.contains('dark');
      const chartStats = compactTrendStats(hourlyStats);
      const isCompact = chartStats.length < hourlyStats.length;
      const isMinuteLevel = detectMinuteLevel(chartStats);
      const labels = chartStats.map(item => formatTimeLabel(item.hour, isMinuteLevel));

      trendChartInstance.setOption({
        animationDuration: 500,
        grid: { left: 20, right: 20, top: 50, bottom: 20, containLabel: true },
        tooltip: {
          trigger: 'axis',
          backgroundColor: isDark ? 'rgba(15,23,42,0.92)' : 'rgba(255,255,255,0.92)',
          borderColor: isDark ? 'rgba(45,212,191,0.18)' : 'rgba(15,23,42,0.08)',
          borderWidth: 1,
          extraCssText: 'border-radius:14px;box-shadow:0 16px 40px rgba(15,23,42,0.12);',
          textStyle: { color: isDark ? '#e2e8f0' : '#0f172a' },
        },
        legend: {
          top: 0,
          icon: 'roundRect',
          itemWidth: 12,
          itemHeight: 8,
          textStyle: { color: isDark ? '#cbd5e1' : '#475569', fontSize: 12 },
        },
        xAxis: {
          type: 'category', data: labels,
          axisLabel: { show: false },
          axisLine: { lineStyle: { color: isDark ? '#334155' : '#cbd5e1' } },
          axisTick: { show: false },
        },
        yAxis: [
          {
            type: 'value', name: '请求数',
            axisLabel: { color: isDark ? '#94a3b8' : '#64748b' },
            splitLine: { lineStyle: { color: isDark ? 'rgba(148,163,184,0.10)' : 'rgba(148,163,184,0.18)' } },
          },
          {
            type: 'value', name: '耗时(s)',
            axisLabel: { color: isDark ? '#94a3b8' : '#64748b' },
            splitLine: { show: false },
          },
        ],
        series: [
          {
            name: '总请求', type: 'bar', barMaxWidth: 18, barCategoryGap: '48%',
            itemStyle: {
              borderRadius: [8, 8, 0, 0],
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: isDark ? '#5eead4' : '#2dd4bf' },
                { offset: 1, color: isDark ? '#0f766e' : '#0f766e' },
              ]),
            },
            data: chartStats.map(item => item.total),
          },
          {
            name: '成功请求', type: 'line', smooth: true, symbol: 'circle', symbolSize: 5, showSymbol: !isCompact,
            lineStyle: { width: 2.6, color: '#10b981' }, itemStyle: { color: '#10b981' },
            areaStyle: { color: 'rgba(16,185,129,0.12)' },
            data: chartStats.map(item => item.success),
          },
          {
            name: '失败请求', type: 'line', smooth: true, symbol: 'circle', symbolSize: 5, showSymbol: !isCompact,
            lineStyle: { width: 2.4, color: '#f43f5e' }, itemStyle: { color: '#f43f5e' },
            areaStyle: { color: 'rgba(244,63,94,0.10)' },
            data: chartStats.map(item => item.failed),
          },
          {
            name: '平均耗时', type: 'line', smooth: true, symbol: 'circle', symbolSize: 5, showSymbol: !isCompact,
            yAxisIndex: 1,
            lineStyle: { width: 2.2, color: '#f59e0b' }, itemStyle: { color: '#f59e0b' },
            data: chartStats.map(item => Number(item.avgTime || 0)),
          },
        ],
      });
    }

    // ==================== 趋势模型选择器 ====================
    function renderTrendModelOptions(models) {
      state.modelOptions = models;
      const hasCurrent = state.trendModelName === '__all__' || models.some(m => m.name === state.trendModelName);
      if (!hasCurrent) { state.trendModelName = '__all__'; state.trendMode = 'all'; }
      elements.trendModelSelectLabel.textContent = state.trendModelName === '__all__' ? '总趋势' : state.trendModelName;
      renderTrendModelOptionsList('');
    }

    function renderTrendModelOptionsList(keyword = '') {
      const term = keyword.trim().toLowerCase();
      const options = [{ name: '__all__', label: '总趋势' }].concat(
        state.modelOptions.filter(m => !term || m.name.toLowerCase().includes(term)).map(m => ({ name: m.name, label: m.name }))
      );

      elements.trendModelOptions.innerHTML = options.map(option => {
        const isSelected = state.trendModelName === option.name;
        return `<button type="button" class="flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm transition ${isSelected ? 'bg-primary-500/12 text-primary-700 dark:bg-primary-500/18 dark:text-primary-200' : 'text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800/80'}" data-model-option="${option.name}">
          <span class="truncate">${option.label}</span>
          ${isSelected ? '<i data-lucide="check" class="h-4 w-4"></i>' : ''}
        </button>`;
      }).join('') || '<div class="px-3 py-4 text-sm text-slate-500 dark:text-slate-400">没有匹配的模型</div>';

      elements.trendModelOptions.querySelectorAll('[data-model-option]').forEach(button => {
        button.addEventListener('click', () => {
          state.trendModelName = button.getAttribute('data-model-option') || '__all__';
          state.trendMode = state.trendModelName === '__all__' ? 'all' : 'model';
          elements.trendModelSelectLabel.textContent = state.trendModelName === '__all__' ? '总趋势' : state.trendModelName;
          elements.trendModelDropdown.classList.add('hidden');
          elements.trendModelSearchInput.value = '';
          renderTrendModelOptionsList('');
          hideError();
          loadTrendBySelection();
          lucide.createIcons();
        });
      });

      lucide.createIcons();
    }

    function renderTrendModelCards(summary) {
      if (!summary) {
        elements.trendModelCards.classList.add('hidden');
        elements.trendModelCards.innerHTML = '';
        return;
      }

      const cards = [
        { label: '模型请求数', value: formatNumber(summary.totalRecords), hint: `成功 ${formatNumber(summary.successRecords)} / 失败 ${formatNumber(summary.failedRecords)}`, icon: 'activity' },
        { label: '模型成功率', value: formatPercent(summary.successRate), hint: `平均耗时 ${formatSeconds(summary.avgTime)}`, icon: 'shield-check' },
        { label: '模型消耗额度', value: formatCreditsFromQuota(summary.totalQuota), hint: `总 Tokens ${formatNumber(summary.totalTokens)}`, icon: 'wallet-cards' },
        { label: '活跃时段', value: formatNumber(summary.activeHours), hint: `峰值请求 ${formatNumber(summary.peakCount)}`, icon: 'clock-3' },
      ];

      elements.trendModelCards.innerHTML = cards.map(card => `
        <article class="metric-chip rounded-2xl px-4 py-4">
          <div class="flex items-start justify-between gap-3">
            <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">${card.label}</div>
            <span class="section-icon shrink-0"><i data-lucide="${card.icon}" class="h-4 w-4"></i></span>
          </div>
          <div class="mt-3 text-2xl font-black text-slate-900 dark:text-white">${card.value}</div>
          <div class="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">${card.hint}</div>
        </article>
      `).join('');
      elements.trendModelCards.classList.remove('hidden');
    }

    async function loadTrendBySelection() {
      if (!state.data) return;

      if (state.trendModelName === '__all__') {
        state.trendMode = 'all';
        state.trendModelData = null;
        renderTrendModelCards(null);
        renderTrend(state.data.hourly_stats, null);
        return;
      }

      state.trendMode = 'model';
      // 这里调用 /api/logs 获取模型趋势数据，后端需要实现
      // 如果没有后端，展示占位数据
      try {
        const response = await fetchAPI(`/api/logs?model_name=${encodeURIComponent(state.trendModelName)}&hours=${state.hours}`);
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(payload.message || `请求失败 (${response.status})`);

        state.trendModelData = payload.data;
        renderTrendModelCards(payload.data.summary);
        renderTrend(payload.data.hourly_stats || [], payload.data);
      } catch (error) {
        showError(error instanceof Error ? error.message : '模型趋势加载失败');
      } finally {
        lucide.createIcons();
      }
    }

    // ==================== 热门模型排行 ====================
    function setTopModelsView(view) {
      state.topModelsView = view;
      const isTable = view === 'table';

      elements.topModelsTableView.classList.toggle('hidden', !isTable);
      elements.topModelsChartView.classList.toggle('hidden', isTable);

      const activeClass = 'inline-flex items-center gap-2 rounded-xl bg-primary-500 px-3 py-2 text-xs font-bold text-white transition hover:bg-primary-600 dark:bg-primary-500 dark:text-slate-950 dark:hover:bg-primary-400';
      const inactiveClass = 'inline-flex items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold text-slate-600 transition hover:text-slate-900 dark:text-slate-300 dark:hover:text-white';

      document.getElementById('topModelsTableViewButton').className = isTable ? activeClass : inactiveClass;
      document.getElementById('topModelsChartViewButton').className = isTable ? inactiveClass : activeClass;

      if (!isTable && state.data?.top_models?.length) {
        renderTopModelsChart(state.data.top_models);
      }

      lucide.createIcons();
    }

    function renderTopModels(topModels) {
      if (!topModels.length) {
        elements.topModelsTable.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">暂无模型排行数据</td></tr>';
        if (topModelsChartInstance) topModelsChartInstance.clear();
        return;
      }

      elements.topModelsTable.innerHTML = topModels.map(model => {
        const isSelected = state.trendMode === 'model' && state.trendModelName === model.name;
        const isExpanded = state.expandedModelName === model.name;
        return `
          <tr class="table-row rounded-2xl text-sm text-slate-700 dark:text-slate-200 ${isSelected ? 'selected-model-row' : 'bg-white/70 dark:bg-slate-900/70'}">
            <td class="rounded-l-2xl px-4 py-4">
              <button type="button" class="flex w-full items-center gap-3 text-left" data-model-row="${model.name}">
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-primary-500/10 text-primary-600 dark:bg-primary-500/20 dark:text-primary-300">
                  <i data-lucide="${isExpanded ? 'chevron-down' : 'chevron-right'}" class="h-4 w-4"></i>
                </span>
                <div class="min-w-0">
                  <div class="truncate font-mono text-base font-bold text-slate-800 dark:text-white">${model.name}</div>
                  ${isSelected ? '<div class="mt-1 text-[11px] font-bold text-primary-600 dark:text-primary-300">当前趋势模型</div>' : ''}
                </div>
              </button>
            </td>
            <td class="px-4 py-4 font-semibold tabular-nums">${formatNumber(model.count)}</td>
            <td class="px-4 py-4 font-semibold tabular-nums ${successRateClass(model.successRate)}">${formatPercent(model.successRate)}</td>
            <td class="px-4 py-4 tabular-nums">${formatSeconds(model.avgTime)}</td>
            <td class="px-4 py-4 tabular-nums">${formatNumber(model.totalTokens)}</td>
            <td class="rounded-r-2xl px-4 py-4 tabular-nums">${formatCreditsFromQuota(model.totalQuota)}</td>
          </tr>
          ${isExpanded ? `
            <tr>
              <td colspan="6" class="px-2 pt-0">
                <div class="rounded-2xl border border-slate-200/70 bg-slate-50/80 px-4 py-4 dark:border-slate-700 dark:bg-slate-900/70">
                  <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">最近使用</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatTimestamp(model.lastUsedAt)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">输入 Tokens</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatNumber(model.totalPromptTokens)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">输出 Tokens</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatNumber(model.totalCompletionTokens)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">失败次数</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatNumber(model.failedCount)}</div>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          ` : ''}
        `;
      }).join('');

      elements.topModelsTable.querySelectorAll('[data-model-row]').forEach(button => {
        button.addEventListener('click', () => {
          const modelName = button.getAttribute('data-model-row');
          state.expandedModelName = state.expandedModelName === modelName ? null : modelName;
          renderTopModels(topModels);
          lucide.createIcons();
        });
      });

      lucide.createIcons();
    }

    function renderTopModelsChart(topModels) {
      if (!topModels.length) {
        if (topModelsChartInstance) topModelsChartInstance.clear();
        return;
      }

      if (!topModelsChartInstance) {
        topModelsChartInstance = echarts.init(elements.topModelsChartView, null, { renderer: 'canvas' });
        ensureChartResizeListener();
      }

      const isDark = document.documentElement.classList.contains('dark');
      const models = topModels.slice(0, 6);
      const radarColors = ['#14b8a6', '#10b981', '#f59e0b', '#38bdf8', '#f43f5e', '#6366f1'];
      const maxCount = Math.max(...models.map(m => Number(m.count || 0)), 1);
      const maxTokens = Math.max(...models.map(m => Number(m.totalTokens || 0)), 1);
      const maxCredits = Math.max(...models.map(m => Number(m.totalQuota || 0) / quotaPerUnit), 1);
      const maxActiveHours = Math.max(...models.map(m => Number(m.activeHours || 0)), 1);
      const maxAvgTime = Math.max(...models.map(m => Number(m.avgTime || 0)), 1);

      const radarIndicators = [
        { name: '请求量', max: 100 },
        { name: '成功率', max: 100 },
        { name: '响应速度', max: 100 },
        { name: '总 Tokens', max: 100 },
        { name: '消耗额度', max: 100 },
        { name: '活跃时段', max: 100 },
      ];

      const radarSeriesData = models.map(model => {
        const credits = Number(model.totalQuota || 0) / quotaPerUnit;
        const avgTime = Number(model.avgTime || 0);
        const speedScore = maxAvgTime > 0 ? Math.max(0, 100 - (avgTime / maxAvgTime) * 100) : 100;
        return {
          name: model.name,
          value: [
            Number(((Number(model.count || 0) / maxCount) * 100).toFixed(2)),
            Number(model.successRate || 0),
            Number(speedScore.toFixed(2)),
            Number(((Number(model.totalTokens || 0) / maxTokens) * 100).toFixed(2)),
            Number(((credits / maxCredits) * 100).toFixed(2)),
            Number(((Number(model.activeHours || 0) / maxActiveHours) * 100).toFixed(2)),
          ],
          raw: model,
        };
      });

      topModelsChartInstance.setOption({
        animationDuration: 500,
        tooltip: {
          trigger: 'item',
          backgroundColor: isDark ? 'rgba(15,23,42,0.94)' : 'rgba(255,255,255,0.96)',
          borderColor: isDark ? 'rgba(45,212,191,0.18)' : 'rgba(15,23,42,0.08)',
          textStyle: { color: isDark ? '#e2e8f0' : '#0f172a' },
          formatter: (params) => {
            const raw = params.data.raw;
            return [
              `<div style="font-weight:700;margin-bottom:6px;">${raw.name}</div>`,
              `请求数：${formatNumber(raw.count)}`,
              `成功率：${formatPercent(raw.successRate)}`,
              `平均耗时：${formatSeconds(raw.avgTime)}`,
              `总 Tokens：${formatNumber(raw.totalTokens)}`,
              `消耗额度：${formatCreditsFromQuota(raw.totalQuota)}`,
              `活跃时段：${formatNumber(raw.activeHours)}`,
            ].join('<br/>');
          },
        },
        legend: {
          top: 'middle', right: 0, orient: 'vertical',
          textStyle: { color: isDark ? '#cbd5e1' : '#475569', fontSize: 12, width: 170, overflow: 'truncate' },
          type: 'scroll', data: models.map(m => m.name),
        },
        radar: {
          center: ['34%', '56%'], radius: '58%',
          indicator: radarIndicators, splitNumber: 5,
          axisName: { color: isDark ? '#e2e8f0' : '#334155', fontSize: 12, fontWeight: 700 },
          axisLine: { lineStyle: { color: isDark ? 'rgba(148,163,184,0.18)' : 'rgba(148,163,184,0.26)' } },
          splitLine: { lineStyle: { color: isDark ? 'rgba(148,163,184,0.14)' : 'rgba(148,163,184,0.18)' } },
          splitArea: { areaStyle: { color: isDark ? ['rgba(15,23,42,0.12)', 'rgba(15,23,42,0.04)'] : ['rgba(255,255,255,0.12)', 'rgba(255,255,255,0.04)'] } },
        },
        color: radarColors,
        series: [{
          type: 'radar', symbol: 'circle', symbolSize: 6,
          data: radarSeriesData,
          lineStyle: { width: 2.5 },
          itemStyle: { borderWidth: 2, borderColor: isDark ? '#0f172a' : '#ffffff' },
          areaStyle: { opacity: 0.14 },
          emphasis: { lineStyle: { width: 3.5 }, areaStyle: { opacity: 0.20 } },
        }],
      });
    }

    // ==================== Key/渠道详情 ====================
    function parseDetailSearchQuery(rawValue) {
      const value = rawValue.trim();
      if (!value) return null;
      if (/^sk-/i.test(value)) return { mode: 'key', value };
      if (/^\d+$/.test(value)) return { mode: 'channel', value };
      return { mode: 'key', value };
    }

    function clearDetailInfo() {
      state.detailData = null;
      state.detailMode = null;
      state.detailExpandedModelName = null;
      elements.keyInfoSection.classList.add('hidden');
      elements.dashboardContent.classList.remove('hidden');
      elements.keyInfoCards.innerHTML = '';
      elements.detailMetaGrid.innerHTML = '';
      elements.detailMetaGrid.classList.add('hidden');
      elements.keyModelsTable.innerHTML = '';
      if (keyTrendChartInstance) keyTrendChartInstance.clear();
      document.getElementById('detailTrendEyebrow').textContent = 'Key Trend';
      document.getElementById('detailTrendTitle').textContent = 'Key 使用趋势';
      document.getElementById('detailModelsEyebrow').textContent = 'Key Models';
      document.getElementById('detailModelsTitle').textContent = 'Key 模型排行';
    }

    function renderKeyTrendChart(hourlyStats) {
      if (!hourlyStats.length) {
        if (keyTrendChartInstance) keyTrendChartInstance.clear();
        elements.keyTrendChart.innerHTML = '<div class="flex h-full items-center justify-center rounded-2xl border border-dashed border-slate-300 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">暂无趋势数据</div>';
        return;
      }

      if (keyTrendChartInstance) keyTrendChartInstance.dispose();
      keyTrendChartInstance = echarts.init(elements.keyTrendChart, null, { renderer: 'canvas' });
      ensureChartResizeListener();

      const isDark = document.documentElement.classList.contains('dark');
      const chartStats = compactTrendStats(hourlyStats);
      const isCompact = chartStats.length < hourlyStats.length;
      const isMinute = detectMinuteLevel(chartStats);
      const labels = chartStats.map(item => formatTimeLabel(item.hour, isMinute));

      keyTrendChartInstance.setOption({
        animationDuration: 500,
        grid: { left: 20, right: 20, top: 50, bottom: 20, containLabel: true },
        tooltip: {
          trigger: 'axis',
          backgroundColor: isDark ? 'rgba(15,23,42,0.94)' : 'rgba(255,255,255,0.96)',
          borderColor: isDark ? 'rgba(45,212,191,0.18)' : 'rgba(15,23,42,0.08)',
          textStyle: { color: isDark ? '#e2e8f0' : '#0f172a' },
        },
        legend: { top: 0, textStyle: { color: isDark ? '#cbd5e1' : '#475569', fontSize: 12 } },
        xAxis: {
          type: 'category', data: labels,
          axisLabel: { show: false },
          axisLine: { lineStyle: { color: isDark ? '#334155' : '#cbd5e1' } },
          axisTick: { show: false },
        },
        yAxis: [{
          type: 'value', name: '请求数',
          axisLabel: { color: isDark ? '#94a3b8' : '#64748b' },
          splitLine: { lineStyle: { color: isDark ? 'rgba(148,163,184,0.10)' : 'rgba(148,163,184,0.18)' } },
        }],
        series: [
          {
            name: '总请求', type: 'bar', barMaxWidth: 20, barCategoryGap: '44%',
            itemStyle: {
              borderRadius: [7, 7, 0, 0],
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: '#2dd4bf' }, { offset: 1, color: '#0d9488' },
              ]),
            },
            data: chartStats.map(item => item.total),
          },
          {
            name: '成功请求', type: 'line', smooth: true, symbol: 'circle', symbolSize: 5, showSymbol: !isCompact,
            lineStyle: { width: 2.6, color: '#10b981' }, itemStyle: { color: '#10b981' },
            areaStyle: { color: 'rgba(16,185,129,0.12)' },
            data: chartStats.map(item => item.success),
          },
          {
            name: '失败请求', type: 'line', smooth: true, symbol: 'circle', symbolSize: 5, showSymbol: !isCompact,
            lineStyle: { width: 2.4, color: '#f43f5e' }, itemStyle: { color: '#f43f5e' },
            areaStyle: { color: 'rgba(244,63,94,0.10)' },
            data: chartStats.map(item => item.failed),
          },
        ],
      });
    }

    function renderKeyModelsTable(topModels) {
      if (!topModels.length) {
        elements.keyModelsTable.innerHTML = '<tr><td colspan="6" class="px-4 py-8 text-center text-sm text-slate-500 dark:text-slate-400">暂无模型数据</td></tr>';
        return;
      }

      elements.keyModelsTable.innerHTML = topModels.map(model => {
        const isExpanded = state.detailExpandedModelName === model.name;
        return `
          <tr class="table-row rounded-2xl bg-white/70 text-sm text-slate-700 dark:bg-slate-900/70 dark:text-slate-200">
            <td class="rounded-l-2xl px-4 py-4">
              <button type="button" class="flex w-full items-center gap-3 text-left" data-detail-model-row="${model.name}">
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-primary-500/10 text-primary-600 dark:bg-primary-500/20 dark:text-primary-300">
                  <i data-lucide="${isExpanded ? 'chevron-down' : 'chevron-right'}" class="h-4 w-4"></i>
                </span>
                <div class="min-w-0 font-mono text-base font-bold text-slate-800 dark:text-white">${model.name}</div>
              </button>
            </td>
            <td class="px-4 py-4 font-semibold tabular-nums">${formatNumber(model.count)}</td>
            <td class="px-4 py-4 font-semibold tabular-nums ${successRateClass(model.successRate)}">${formatPercent(model.successRate)}</td>
            <td class="px-4 py-4 tabular-nums">${formatSeconds(model.avgTime)}</td>
            <td class="px-4 py-4 tabular-nums">${formatNumber(model.totalTokens)}</td>
            <td class="rounded-r-2xl px-4 py-4 tabular-nums">${formatCreditsFromQuota(model.totalQuota)}</td>
          </tr>
          ${isExpanded ? `
            <tr>
              <td colspan="6" class="px-2 pt-0">
                <div class="rounded-2xl border border-slate-200/70 bg-slate-50/80 px-4 py-4 dark:border-slate-700 dark:bg-slate-900/70">
                  <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">成功 / 失败</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatNumber(model.successCount)} / ${formatNumber(model.failedCount)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">首次 / 最近使用</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatTimestamp(model.firstUsedAt)}</div>
                      <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">${formatTimestamp(model.lastUsedAt)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">输入 / 输出 Tokens</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatNumber(model.totalPromptTokens)}</div>
                      <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">${formatNumber(model.totalCompletionTokens)}</div>
                    </div>
                    <div class="metric-chip rounded-xl px-3 py-3">
                      <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">平均消耗</div>
                      <div class="mt-1.5 text-sm font-bold text-slate-900 dark:text-white">${formatCreditsFromQuota(model.avgQuotaPerRequest)}</div>
                      <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">${formatNumber(model.avgTotalTokens)} Tokens / 次</div>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          ` : ''}
        `;
      }).join('');

      elements.keyModelsTable.querySelectorAll('[data-detail-model-row]').forEach(button => {
        button.addEventListener('click', () => {
          const modelName = button.getAttribute('data-detail-model-row');
          state.detailExpandedModelName = state.detailExpandedModelName === modelName ? null : modelName;
          renderKeyModelsTable(topModels);
          lucide.createIcons();
        });
      });

      lucide.createIcons();
    }

    // ==================== 数据加载 ====================
    function renderDashboard(data) {
      state.data = data;
      const lastUpdatedChip = document.getElementById('lastUpdatedChip');
      if (lastUpdatedChip) {
        lastUpdatedChip.textContent = `更新时间 ${new Date().toLocaleString('zh-CN', { hour12: false })}`;
      }
      renderOverview(data.overview);
      renderTopModels(data.top_models);
      renderTrendModelOptions(state.modelOptions.length ? state.modelOptions : (data.top_models || []));
      if (state.topModelsView === 'chart') {
        renderTopModelsChart(data.top_models);
      }
      loadTrendBySelection();
    }

    async function loadDashboard() {
      hideError();
      setLoading(true);

      try {
        // 同时加载仪表盘数据和模型选项
        const [dashboardRes, modelOptions] = await Promise.all([
          fetchAPI(`/api/dashboard?hours=${state.hours}`),
          loadModelOptions(state.hours),
        ]);

        const payload = await dashboardRes.json().catch(() => ({}));
        if (!dashboardRes.ok) {
          throw new Error(payload.message || `请求失败 (${dashboardRes.status})`);
        }

        state.modelOptions = Array.isArray(modelOptions) && modelOptions.length
          ? modelOptions
          : (payload.data?.top_models || []);
        renderDashboard(payload.data);
      } catch (error) {
        showError(error instanceof Error ? error.message : '请求失败');
      } finally {
        setLoading(false);
        lucide.createIcons();
      }
    }

    async function loadModelOptions(hours) {
      try {
        const response = await fetchAPI(`/api/logs/models?hours=${hours}`);
        const payload = await response.json().catch(() => ({}));
        if (!response.ok || !payload.data) return null;
        return payload.data;
      } catch (_) { return null; }
    }

    async function loadDetailBySearch(rawValue) {
      const parsed = parseDetailSearchQuery(rawValue);
      if (!parsed) { clearDetailInfo(); return; }

      setLoading(true);
      try {
        let endpoint, params;
        if (parsed.mode === 'channel') {
          endpoint = '/api/channel/records';
          params = { channel_id: parsed.value, hours: state.hours };
        } else {
          endpoint = '/api/key/quota';
          params = { key: parsed.value, hours: state.hours };
        }

        const queryParams = new URLSearchParams(params);
        const response = await fetchAPI(`${endpoint}?${queryParams}`);
        const payload = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(payload.message || `查询失败 (${response.status})`);

        // 渲染详情
        const data = payload.data;
        if (parsed.mode === 'channel') {
          renderChannelInfo(data);
        } else {
          renderKeyInfo(data);
        }
      } catch (error) {
        showError(error instanceof Error ? error.message : '查询失败');
        clearDetailInfo();
      } finally {
        setLoading(false);
        lucide.createIcons();
      }
    }

    function renderKeyInfo(data) {
      state.detailData = data;
      state.detailMode = 'key';
      state.detailExpandedModelName = null;
      document.getElementById('detailTrendEyebrow').textContent = 'Key Trend';
      document.getElementById('detailTrendTitle').textContent = 'Key 使用趋势';
      document.getElementById('detailModelsEyebrow').textContent = 'Key Models';
      document.getElementById('detailModelsTitle').textContent = 'Key 模型排行';

      const token = data.token;
      const user = data.user;
      const usage = data.usage_summary;
      const statusLabel = formatStatusLabel(token.status);
      const quotaLabel = token.unlimitedQuota ? '无限额度' : `剩余 ${formatCreditsFromQuota(token.remainQuota)}`;
      const totalQuota = Number(token.usedQuota || 0) + Number(token.remainQuota || 0);
      const shouldShowQuotaProgress = !token.unlimitedQuota && totalQuota > 0;
      const quotaUsagePercent = shouldShowQuotaProgress
        ? Math.min(100, Math.max(0, (Number(token.usedQuota || 0) / totalQuota) * 100))
        : 0;

      elements.keyInfoCards.innerHTML = `
        <article class="glass metric-card rounded-[28px] p-6 md:p-7 xl:col-span-4">
          <div class="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.9fr)] xl:items-start">
            <div class="flex h-full min-h-[280px] flex-col">
              <div class="flex items-center gap-3">
                <span class="section-icon shrink-0"><i data-lucide="key-round" class="h-4 w-4"></i></span>
                <div class="text-xs font-extrabold uppercase tracking-[0.22em] text-primary-700 dark:text-primary-300">Key 信息</div>
              </div>
              <div class="flex flex-1 flex-col justify-center">
                <div class="text-3xl font-black tracking-tight text-slate-900 dark:text-white md:text-4xl">${token.name}</div>
                <div class="mt-3 text-sm leading-7 text-slate-600 dark:text-slate-300">${token.maskedKey} · ${formatNullableText(token.group)} · ${statusLabel}</div>
                ${shouldShowQuotaProgress ? `
                <div class="mt-5 max-w-xl">
                  <div class="mb-2 flex items-center justify-between gap-4 text-xs font-bold text-slate-500 dark:text-slate-400">
                    <span>使用进度 ${quotaUsagePercent.toFixed(2)}%</span>
                    <span>${formatCreditsFromQuota(token.usedQuota)} / ${formatCreditsFromQuota(totalQuota)}</span>
                  </div>
                  <div class="h-2.5 overflow-hidden rounded-full bg-slate-200/80 dark:bg-slate-800/80">
                    <div class="h-full rounded-full bg-gradient-to-r from-primary-500 via-cyan-500 to-emerald-500" style="width: ${quotaUsagePercent.toFixed(2)}%"></div>
                  </div>
                </div>
                ` : ''}
                <div class="mt-5 flex flex-wrap gap-2">
                  <span class="metric-chip rounded-full px-3 py-1.5 text-xs font-bold text-slate-600 dark:text-slate-300">用户 ${user.username}</span>
                  <span class="metric-chip rounded-full px-3 py-1.5 text-xs font-bold text-slate-600 dark:text-slate-300">${formatNullableText(user.displayName)}</span>
                  <span class="metric-chip rounded-full px-3 py-1.5 text-xs font-bold text-slate-600 dark:text-slate-300">成功率 ${formatPercent(usage.successRate)}</span>
                  <span class="metric-chip rounded-full px-3 py-1.5 text-xs font-bold text-slate-600 dark:text-slate-300">${formatNumber(usage.modelCount)} 个模型</span>
                </div>
              </div>
            </div>
            <div class="rounded-[24px] border border-primary-400/30 bg-gradient-to-br from-primary-600 via-cyan-600 to-emerald-600 p-5 text-white shadow-glow">
              <div class="text-xs font-extrabold uppercase tracking-[0.18em] text-white/80">额度状态</div>
              <div class="mt-3 text-3xl font-black text-white">${quotaLabel}</div>
              <div class="mt-2 text-sm leading-7 text-white/85">已用 ${formatCreditsFromQuota(token.usedQuota)} · 累计请求 ${formatNumber(user.requestCount)} 次</div>
              <div class="mt-5 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-4 rounded-2xl border border-white/20 bg-white/15 px-4 py-3">
                  <span class="font-bold text-white/80">Key 汇总</span>
                  <span class="font-black text-white">${formatNumber(usage.totalRecords)} 次</span>
                </div>
                <div class="flex items-center justify-between gap-4 rounded-2xl border border-white/20 bg-white/15 px-4 py-3">
                  <span class="font-bold text-white/80">总 Tokens</span>
                  <span class="font-black text-white">${formatNumber(usage.totalTokens)}</span>
                </div>
                <div class="flex items-center justify-between gap-4 rounded-2xl border border-white/20 bg-white/15 px-4 py-3">
                  <span class="font-bold text-white/80">当前状态</span>
                  <span class="font-black text-white">${statusLabel}</span>
                </div>
              </div>
            </div>
          </div>
          <div class="mt-6 grid gap-4 xl:grid-cols-2">
            <div class="metric-chip rounded-[24px] px-5 py-5">
              <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">时间信息</div>
              <div class="mt-4 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">创建时间</span>
                  <span class="font-bold text-slate-900 dark:text-white">${formatTimestamp(token.createdTime)}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">最近访问</span>
                  <span class="font-bold text-slate-900 dark:text-white">${formatTimestamp(token.accessedTime)}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">过期时间</span>
                  <span class="font-bold text-slate-900 dark:text-white">${Number(token.expiredTime) === -1 ? '不过期' : formatTimestamp(token.expiredTime)}</span>
                </div>
              </div>
            </div>
            <div class="metric-chip rounded-[24px] px-5 py-5">
              <div class="text-xs font-extrabold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">用户信息</div>
              <div class="mt-4 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">用户 ID</span>
                  <span class="font-bold text-slate-900 dark:text-white">${formatNullableText(user.userId)}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">用户额度</span>
                  <span class="font-bold text-slate-900 dark:text-white">${formatCreditsFromQuota(user.remainQuota)}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-slate-500 dark:text-slate-400">累计已用</span>
                  <span class="font-bold text-slate-900 dark:text-white">${formatCreditsFromQuota(user.usedQuota)}</span>
                </div>
              </div>
            </div>
          </div>
        </article>
      `;

      elements.detailMetaGrid.innerHTML = '';
      elements.detailMetaGrid.classList.add('hidden');
      renderKeyModelsTable(data.top_models || []);
      elements.dashboardContent.classList.add('hidden');
      elements.keyInfoSection.classList.remove('hidden');
      requestAnimationFrame(() => { renderKeyTrendChart(data.hourly_stats || []); lucide.createIcons(); });
    }

    function renderChannelInfo(data) {
      state.detailData = data;
      state.detailMode = 'channel';
      state.detailExpandedModelName = null;
      document.getElementById('detailTrendEyebrow').textContent = 'Channel Trend';
      document.getElementById('detailTrendTitle').textContent = '渠道使用趋势';
      document.getElementById('detailModelsEyebrow').textContent = 'Channel Models';
      document.getElementById('detailModelsTitle').textContent = '渠道模型排行';

      const channel = data.channel;
      const usage = data.usage_summary;
      const statusLabel = formatStatusLabel(channel.status);
      const autoBanLabel = Number(channel.autoBan) === 1 ? '开启' : '关闭';
      const tagLabel = channel.tag ? channel.tag : '无标签';

      const cards = [
        { label: '渠道信息', value: `#${channel.channelId}`, hint: `类型 ${formatNumber(channel.channelType)} · ${statusLabel}`, icon: 'waypoints' },
        { label: '响应与消耗', value: `${formatNumber(channel.responseTime)} ms`, hint: `已用 ${formatCreditsFromQuota(channel.usedQuota)} · 余额 ${formatNumber(channel.balance)}`, icon: 'gauge' },
        { label: '路由配置', value: `优先级 ${formatNumber(channel.priority)}`, hint: `权重 ${formatNumber(channel.weight)} · 自动封禁 ${autoBanLabel} · ${tagLabel}`, icon: 'git-branch-plus' },
        { label: '渠道汇总', value: `${formatNumber(usage.totalRecords)} 次`, hint: `成功率 ${formatPercent(usage.successRate)} · 失败 ${formatNumber(usage.failedRecords)}`, icon: 'bar-chart-3' },
      ];

      elements.keyInfoCards.innerHTML = cards.map(card => `
        <article class="glass metric-card rounded-[24px] p-6">
          <div class="flex items-start justify-between gap-4">
            <div class="text-xs font-extrabold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">${card.label}</div>
            <span class="section-icon shrink-0"><i data-lucide="${card.icon}" class="h-4 w-4"></i></span>
          </div>
          <div class="mt-4 text-2xl font-black text-slate-900 dark:text-white">${card.value}</div>
          <div class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">${card.hint}</div>
        </article>
      `).join('');

      // 详情元数据
      const detailMetaGrid = elements.detailMetaGrid;
      const metaItems = [
        { label: '创建时间', value: formatTimestamp(channel.createdTime), hint: `最近测试 ${formatTimestamp(channel.testTime)}` },
        { label: '平均耗时', value: formatSeconds(usage.avgTime), hint: `成功率 ${formatPercent(usage.successRate)}` },
        { label: '历史额度', value: formatCreditsFromQuota(usage.totalQuota), hint: `总 Tokens ${formatNumber(usage.totalTokens)}` },
        { label: '输入 Tokens', value: formatNumber(usage.totalPromptTokens), hint: `输出 ${formatNumber(usage.totalCompletionTokens)}` },
      ];

      detailMetaGrid.innerHTML = metaItems.map(item => `
        <article class="glass metric-card rounded-[24px] p-6">
          <div class="text-xs font-extrabold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">${item.label}</div>
          <div class="mt-4 text-2xl font-black text-slate-900 dark:text-white">${item.value}</div>
          <div class="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">${item.hint || '&nbsp;'}</div>
        </article>
      `).join('');
      detailMetaGrid.classList.remove('hidden');

      renderKeyModelsTable(data.top_models || []);
      elements.dashboardContent.classList.add('hidden');
      elements.keyInfoSection.classList.remove('hidden');
      requestAnimationFrame(() => { renderKeyTrendChart(data.hourly_stats || []); lucide.createIcons(); });
    }

    function refreshCurrentView() {
      hideError();
      if (state.detailData && elements.keySearchInput.value.trim()) {
        loadDetailBySearch(elements.keySearchInput.value);
        return;
      }
      loadDashboard();
    }

    // ==================== 初始化 ====================
    function initialize() {
      applyTheme();
      bindScrollPerfHints();

      // 主题切换
      document.getElementById('themeToggle').addEventListener('click', toggleTheme);
      elements.refreshButton.addEventListener('click', refreshCurrentView);

      // Key 搜索
      function doKeySearch() {
        hideError();
        const searchValue = elements.keySearchInput.value.trim();
        if (!searchValue) { clearDetailInfo(); return; }
        loadDetailBySearch(searchValue);
      }

      function updateClearButton() {
        elements.keyClearButton.classList.toggle('hidden', !elements.keySearchInput.value.trim());
      }

      elements.keySearchInput.addEventListener('keydown', event => { if (event.key === 'Enter') doKeySearch(); });
      elements.keySearchInput.addEventListener('input', () => {
        updateClearButton();
        if (!elements.keySearchInput.value.trim()) clearDetailInfo();
      });
      elements.keySearchButton.addEventListener('click', doKeySearch);
      elements.keyClearButton.addEventListener('click', () => {
        elements.keySearchInput.value = '';
        updateClearButton();
        clearDetailInfo();
        hideError();
      });

      // 趋势模型下拉
      elements.trendModelSelectButton.addEventListener('click', () => {
        elements.trendModelDropdown.classList.toggle('hidden');
        if (!elements.trendModelDropdown.classList.contains('hidden')) {
          elements.trendModelSearchInput.focus();
          renderTrendModelOptionsList(elements.trendModelSearchInput.value.trim());
        }
        lucide.createIcons();
      });
      elements.trendModelSearchInput.addEventListener('input', () => {
        renderTrendModelOptionsList(elements.trendModelSearchInput.value.trim());
      });
      elements.trendModelSearchInput.addEventListener('keydown', event => {
        if (event.key === 'Escape') {
          elements.trendModelDropdown.classList.add('hidden');
          elements.trendModelSearchInput.blur();
        }
      });
      document.addEventListener('click', event => {
        if (!elements.trendModelDropdown.contains(event.target) && !elements.trendModelSelectButton.contains(event.target)) {
          elements.trendModelDropdown.classList.add('hidden');
        }
      });

      // 视图切换
      document.getElementById('topModelsTableViewButton').addEventListener('click', () => setTopModelsView('table'));
      document.getElementById('topModelsChartViewButton').addEventListener('click', () => setTopModelsView('chart'));
      setTopModelsView('table');

      // 时间范围按钮
      renderHoursButtons();

      // Footer 年份
      document.getElementById('footerYear').textContent = String(new Date().getFullYear());

      // 加载配置
      fetchAPI('/api/config')
        .then(r => r.json())
        .then(payload => {
          const d = payload?.data || {};
          if (d.quotaPerUnit) quotaPerUnit = Number(d.quotaPerUnit);
          if (d.logo) {
            document.getElementById('headerLogo').src = d.logo;
            document.getElementById('footerLogo').src = d.logo;
            document.getElementById('favicon').href = d.logo;
          }
          if (d.systemName) {
            document.getElementById('headerSiteName').textContent = d.systemName;
            document.getElementById('footerName').textContent = d.systemName;
            document.getElementById('footerCopyright').textContent = d.systemName;
            document.title = d.systemName + ' 实时监控台';
          }
          if (d.docsLink) {
            const docsEl = document.getElementById('headerDocsLink');
            docsEl.href = d.docsLink;
            docsEl.classList.remove('hidden');
            docsEl.classList.add('inline-flex');
          }
          if (d.serverAddress) {
            const platformLink = document.getElementById('headerPlatformLink');
            platformLink.href = d.serverAddress;
          }

          // 自动刷新
          const ttl = Number(d.cacheTtlSeconds || 900);
          const interval = Math.round(ttl * 2 / 3) * 1000;
          if (autoRefreshTimer) clearInterval(autoRefreshTimer);
          autoRefreshTimer = setInterval(refreshCurrentView, interval);
        })
        .catch(() => {})
        .finally(() => {
          loadDashboard();
        });

      // 响应系统主题变化
      systemThemeMedia.addEventListener('change', () => {
        if (getThemeMode() === 'auto') {
          applyTheme();
          lucide.createIcons();
        }
      });

      // localStorage 变化
      window.addEventListener('storage', event => {
        if (!event.key || ['theme', 'theme-mode'].includes(event.key)) {
          applyTheme();
          lucide.createIcons();
        }
      });

      lucide.createIcons();
    }

    initialize();
