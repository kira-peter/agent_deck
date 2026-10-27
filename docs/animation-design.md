# 主菜单动画设计

主菜单选中项的 spinner 动画。实现见 `pkg/ui/animation/spinner.go`，共有 **9 种**动画，每次上下切换选项时随机换一种（保证与上一次不同）。

## 动画列表

### 旋转类（7 种）

#### 1. 雪花闪烁（AnimSnowflake，默认）
```
* → ✻ → ❅ → ✼ → ❆ → ❉ → ❆ → ✼ → ❅ → ✻
```
对称渐变，从简单到复杂再回归；启动时的初始动画。

#### 2. Braille 点阵旋转（AnimBraille）
```
⠋ → ⠙ → ⠹ → ⠸ → ⠼ → ⠴ → ⠦ → ⠧ → ⠇ → ⠏
```
经典加载动画，简洁流畅。

#### 3. 箭头旋转（AnimArrow）
```
← → ↖ → ↑ → ↗ → → → ↘ → ↓ → ↙
```
方向感强，动态明显。

#### 4. 月相旋转（AnimMoon）
```
◐ → ◓ → ◑ → ◒
```
圆形渐变，柔和流畅。

#### 5. 方块翻转（AnimSquare）
```
◰ → ◳ → ◲ → ◱
```
方块内部图案旋转，规整对称。

### 移动类（4 种）

#### 6. 跳动的点（AnimDots）
```
⣾ → ⣽ → ⣻ → ⢿ → ⡿ → ⣟ → ⣯ → ⣷
```
Braille 点从左到右移动。

#### 7. 波浪移动（AnimWave）
```
◜ → ◝ → ◞ → ◟
```
圆角方块循环，形成波浪感。

#### 8. 弹跳球（AnimBouncing）
```
˙ → · → ∘ → ○ → ◯ → ○ → ∘ → ·
```
圆点从小到大再到小，模拟弹跳。

#### 9. 蛇形移动（AnimSnake）
```
⠁ → ⠂ → ⠄ → ⡀ → ⢀ → ⠠ → ⠐ → ⠈
```
Braille 点像蛇爬行。

## 切换机制

- **触发时机**：在主菜单按上/下箭头、`j`/`k` 切换选项时
- **选择方式**：`RandomAnimationTypeExcept(current)` 从其余 8 种中随机选一种，保证每次切换都不同
- **初始动画**：`AnimSnowflake`（`NewSpinner()` 的默认值）
- **刷新频率**：`time.Ticker` 每 80ms 一帧（约 12.5 FPS）
- **动画范围**：只有主菜单使用动画；进入具体视图后由 tcell 渲染，不带 spinner

## 技术实现

- 主菜单由 Bubble Tea 驱动，`main.go` 每 80ms 发送一次 `tickMsg` 触发重绘，`Spinner` 内部另有一个 80ms 的 ticker 推进帧
- `Spinner` 用 `sync.RWMutex` 保护帧索引与动画类型，支持运行时切换
- 所有帧字符等宽，切换动画时不会导致菜单左右抖动
- 新增动画：在 `AnimationType` 中加常量、在 `animationFrames` 中加帧序列，并同时加入 `RandomAnimationType` / `RandomAnimationTypeExcept` 的候选列表

> 历史设计稿 [animation-options.md](animation-options.md) 里出现过 10 个候选（含沙漏、时钟等 emoji 动画），因 emoji 宽度不一致会破坏对齐，最终没有采用。
