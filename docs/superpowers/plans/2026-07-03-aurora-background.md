# 极光光晕背景优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将监控台僵硬的硬边斜切背景替换为柔和的极光光晕 + 克制慢速动效（设计文档：`docs/superpowers/specs/2026-07-03-aurora-background-design.md`）。

**架构：** 纯 CSS 改造，全部改动集中在 `index.html` 的 `<style>` 块与背景装饰 div。底色换为无断点平滑渐变；新增 3 个 radial-gradient 光斑并配独立慢速 keyframes；网格加径向遮罩渐隐；删除细斜线；玻璃卡片透明度微调。

**技术栈：** 原生 CSS（radial-gradient / keyframes / mask-image），Go 后端不动。验证用 Playwright MCP 截图 + `go test`。

## 全局约束

- 只修改 `index.html`，禁止改动 Go 代码、页面布局、组件结构、图表配色、交互逻辑
- 不引入任何新依赖
- 背景动画只允许 `transform` 与 `opacity` 两个属性
- 必须保留 `@media (prefers-reduced-motion: reduce)` 停止动画
- 提交信息用简体中文，结尾附 `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`
- `index.html` 由服务端从磁盘 `ServeFile`（`internal/monitor/server.go:305-310`），改完刷新浏览器即可，无需重启服务

---

### Task 1: 背景层重做（底色 + 光斑 + 网格柔化 + 删斜线）

**Files:**
- Modify: `index.html:51-64`（body 背景）
- Modify: `index.html:81-93`（.grid-bg / .signal-bg 样式区）
- Modify: `index.html:182-186`（背景装饰 div）

**Interfaces:**
- Consumes: 现有 `html.dark` 主题切换机制（class 切换），背景装饰容器 `.pointer-events-none.fixed.inset-0.overflow-hidden`
- Produces: `.aurora-blob`、`.aurora-blob--teal/--amber/--blue` 样式类与 `aurora-drift-1/2/3` keyframes（仅本任务内使用，无跨任务接口）

- [ ] **Step 1: 替换 body 背景为平滑渐变**

用 Edit 将：

```css
    body {
      font-family: Manrope, ui-sans-serif, system-ui, sans-serif;
      background:
        linear-gradient(115deg, rgba(8, 153, 131, 0.12) 0 18%, transparent 18% 100%),
        linear-gradient(160deg, transparent 0 64%, rgba(245, 158, 11, 0.10) 64% 76%, transparent 76% 100%),
        linear-gradient(135deg, #f8fafc 0%, #eefcf8 44%, #f8fafc 100%);
    }

    html.dark body {
      background:
        linear-gradient(115deg, rgba(8, 153, 131, 0.18) 0 18%, transparent 18% 100%),
        linear-gradient(160deg, transparent 0 62%, rgba(245, 158, 11, 0.12) 62% 74%, transparent 74% 100%),
        linear-gradient(135deg, #020617 0%, #111827 48%, #030712 100%);
    }
```

替换为：

```css
    body {
      font-family: Manrope, ui-sans-serif, system-ui, sans-serif;
      background: linear-gradient(180deg, #f8fafc 0%, #f0fdfa 50%, #f8fafc 100%);
    }

    html.dark body {
      background: linear-gradient(180deg, #020617 0%, #0f172a 52%, #030712 100%);
    }
```

- [ ] **Step 2: 网格柔化 + 删除 signal-bg + 新增极光光斑样式**

用 Edit 将：

```css
    /* ===== 网格背景 ===== */
    .grid-bg {
      background-image:
        linear-gradient(rgba(8, 153, 131, 0.07) 1px, transparent 1px),
        linear-gradient(90deg, rgba(8, 153, 131, 0.07) 1px, transparent 1px);
      background-size: 56px 56px;
    }

    .signal-bg {
      background-image:
        linear-gradient(120deg, transparent 0 32%, rgba(8, 153, 131, 0.10) 32% 33%, transparent 33% 100%),
        linear-gradient(120deg, transparent 0 68%, rgba(245, 158, 11, 0.12) 68% 69%, transparent 69% 100%);
    }
```

替换为：

```css
    /* ===== 网格背景（径向遮罩，四周渐隐）===== */
    .grid-bg {
      background-image:
        linear-gradient(rgba(8, 153, 131, 0.05) 1px, transparent 1px),
        linear-gradient(90deg, rgba(8, 153, 131, 0.05) 1px, transparent 1px);
      background-size: 56px 56px;
      -webkit-mask-image: radial-gradient(ellipse 75% 60% at 50% 32%, #000 25%, transparent 78%);
      mask-image: radial-gradient(ellipse 75% 60% at 50% 32%, #000 25%, transparent 78%);
    }

    /* ===== 极光光斑 ===== */
    .aurora-blob {
      position: absolute;
      border-radius: 50%;
      will-change: transform, opacity;
    }

    .aurora-blob--teal {
      top: -18vw;
      left: -12vw;
      width: 55vw;
      height: 55vw;
      background: radial-gradient(closest-side, rgba(45, 212, 191, 0.16), transparent);
      animation: aurora-drift-1 75s ease-in-out infinite alternate;
    }

    .aurora-blob--amber {
      top: 30%;
      right: -16vw;
      width: 45vw;
      height: 45vw;
      background: radial-gradient(closest-side, rgba(245, 158, 11, 0.10), transparent);
      animation: aurora-drift-2 90s ease-in-out infinite alternate;
    }

    .aurora-blob--blue {
      bottom: -16vw;
      left: -8vw;
      width: 50vw;
      height: 50vw;
      background: radial-gradient(closest-side, rgba(56, 189, 248, 0.12), transparent);
      animation: aurora-drift-3 105s ease-in-out infinite alternate;
    }

    html.dark .aurora-blob--teal {
      background: radial-gradient(closest-side, rgba(45, 212, 191, 0.22), transparent);
    }

    html.dark .aurora-blob--amber {
      background: radial-gradient(closest-side, rgba(245, 158, 11, 0.14), transparent);
    }

    html.dark .aurora-blob--blue {
      background: radial-gradient(closest-side, rgba(56, 189, 248, 0.16), transparent);
    }

    @keyframes aurora-drift-1 {
      from { transform: translate3d(0, 0, 0) scale(1); }
      to { transform: translate3d(5%, 4%, 0) scale(1.06); }
    }

    @keyframes aurora-drift-2 {
      from { transform: translate3d(0, 0, 0) scale(1.04); opacity: 0.85; }
      to { transform: translate3d(-5%, -3%, 0) scale(1); opacity: 1; }
    }

    @keyframes aurora-drift-3 {
      from { transform: translate3d(0, 0, 0) scale(1); opacity: 1; }
      to { transform: translate3d(4%, -4%, 0) scale(1.05); opacity: 0.8; }
    }

    @media (prefers-reduced-motion: reduce) {
      .aurora-blob { animation: none; }
    }
```

- [ ] **Step 3: 更新背景装饰 HTML（删斜线 div，加 3 个光斑 div）**

用 Edit 将：

```html
  <!-- ===== 背景装饰 ===== -->
  <div class="pointer-events-none fixed inset-0 overflow-hidden">
    <div class="grid-bg absolute inset-0"></div>
    <div class="signal-bg absolute inset-0 opacity-80"></div>
  </div>
```

替换为：

```html
  <!-- ===== 背景装饰 ===== -->
  <div class="pointer-events-none fixed inset-0 overflow-hidden">
    <div class="grid-bg absolute inset-0"></div>
    <div class="aurora-blob aurora-blob--teal"></div>
    <div class="aurora-blob aurora-blob--amber"></div>
    <div class="aurora-blob aurora-blob--blue"></div>
  </div>
```

- [ ] **Step 4: 启动本地服务**

运行（后台）：`PORT=8090 go run ./cmd/server`
预期输出：`monitor server listening on :8090`
（用 8090 避开可能被旧 server.exe 占用的 8080）

- [ ] **Step 5: Playwright 浅色主题验证**

1. `browser_navigate` 到 `http://localhost:8090`
2. `browser_take_screenshot` 全页截图
3. 核对：无硬边斜切色带；左上青绿/右中琥珀/左下冷蓝光斑柔和可见；网格四周渐隐
4. `browser_evaluate` 执行 `() => document.documentElement.scrollWidth <= window.innerWidth`，预期 `true`（无横向滚动）

- [ ] **Step 6: 深色主题验证**

1. `browser_click` 主题切换按钮（`#themeToggle`），必要时点两次直到 `html` 带 `dark` class（可用 `browser_evaluate` 执行 `() => document.documentElement.classList.contains('dark')` 确认）
2. `browser_take_screenshot` 全页截图
3. 核对：深空底色 + 光晕略强、文字可读

- [ ] **Step 7: 提交**

```bash
git add index.html
git commit -m "背景改为极光光晕柔和风格

- body 硬边斜切渐变换为平滑底色
- 新增 3 个慢速漂移呼吸的极光光斑（纯 transform/opacity 动画）
- 网格加径向遮罩四周渐隐，删除细斜线装饰
- 尊重 prefers-reduced-motion

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: 玻璃卡片协调微调

**Files:**
- Modify: `index.html:67-79`（.glass 样式）

**Interfaces:**
- Consumes: Task 1 的新背景（光晕需隐约透过玻璃）
- Produces: 无（终端视觉调整）

- [ ] **Step 1: 调整 .glass 透明度与模糊**

用 Edit 将：

```css
    .glass {
      background: rgba(255, 255, 255, 0.84);
      backdrop-filter: blur(16px);
      -webkit-backdrop-filter: blur(16px);
      border: 1px solid rgba(15, 23, 42, 0.08);
      box-shadow: 0 18px 54px rgba(15, 23, 42, 0.08);
    }

    html.dark .glass {
      background: rgba(15, 23, 42, 0.78);
      border-color: rgba(148, 163, 184, 0.16);
      box-shadow: 0 24px 72px rgba(0, 0, 0, 0.34);
    }
```

替换为：

```css
    .glass {
      background: rgba(255, 255, 255, 0.78);
      backdrop-filter: blur(18px);
      -webkit-backdrop-filter: blur(18px);
      border: 1px solid rgba(15, 23, 42, 0.08);
      box-shadow: 0 18px 54px rgba(15, 23, 42, 0.08);
    }

    html.dark .glass {
      background: rgba(15, 23, 42, 0.72);
      border-color: rgba(148, 163, 184, 0.16);
      box-shadow: 0 24px 72px rgba(0, 0, 0, 0.34);
    }
```

- [ ] **Step 2: 两主题截图核对可读性**

1. 刷新页面（`browser_navigate` 到 `http://localhost:8090`）
2. 浅色截图 → 切深色 → 截图
3. 核对：卡片文字对比度不下降、光晕隐约透过玻璃；若浅色下卡片发灰或文字变淡，把 0.78 回调至 0.80-0.82 后重新截图
4. `browser_evaluate` 再次确认无横向滚动

- [ ] **Step 3: 提交**

```bash
git add index.html
git commit -m "微调玻璃卡片透明度与模糊以配合极光背景

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: 最终验证与收尾

**Files:**
- 无新改动（纯验证；若发现问题回到对应 Task 修复）

**Interfaces:**
- Consumes: Task 1、2 的全部改动
- Produces: 验证结论

- [ ] **Step 1: 后端测试**

运行：`go test ./...`
预期：`ok  realtime-monitor/internal/monitor`（全部通过，无 FAIL）

- [ ] **Step 2: reduced-motion 与动画属性核查**

`browser_evaluate` 执行：

```js
() => {
  const blob = document.querySelector('.aurora-blob');
  const style = getComputedStyle(blob);
  return {
    hasBlobs: document.querySelectorAll('.aurora-blob').length === 3,
    animationName: style.animationName,
    signalRemoved: document.querySelector('.signal-bg') === null,
  };
}
```

预期：`hasBlobs: true`、`animationName` 为 `aurora-drift-1`、`signalRemoved: true`。
同时确认 `index.html` 中存在 `@media (prefers-reduced-motion: reduce)` 规则（Grep 检查）。

- [ ] **Step 3: 收尾**

1. 停止后台 `go run` 服务
2. 关闭 Playwright 浏览器（`browser_close`）
3. 确认 `git status` 干净（除未跟踪的临时截图外无未提交改动）
