# Terminal Visuals

一个零第三方依赖的 Go 模块，提供终端主题、扇形菜单、对话框、边缘渐隐、双线开合、坐标插值、短闪、渐变、扫光、缓动和星点。调用方设置选项文案和行为，并用自己的渲染器绘制。

```go
import visual "github.com/will-to-meaning/terminal_visuals"

palette := visual.Ink()
sweep := visual.Sweep{Duration: 600 * time.Millisecond, Radius: 8}
band := sweep.At(elapsed, width)
color := visual.Mix(palette.Accent, palette.Highlight, band.Strength(x))
```

`visual.Dialog` 提供居中圆角面板、带焦点扫光的三行描边按钮，以及双线展开动画。把标题、正文、按钮文案和当前选项传给 `Dialog.Draw`，通过回调接收坐标、文本与颜色。传入当前渲染器的文字宽度和裁切函数，可正确对齐中文。应用负责按键、按钮行为和帧调度。将 `OpenAge` 或 `FocusAge` 设为负值即可关闭对应动效。

独立仓库发布版本标签后，各项目可以更新模块版本并重新构建。本地联调可用 Go workspace 指向正在编辑的模块。首次发布前，AgentDeck 使用仓库里附带的 `v0.5.0` 源码快照，单独检出 AgentDeck 也能构建。

运行 `go test ./...` 验证这个模块。
