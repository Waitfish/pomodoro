# 番茄钟 (Pomodoro)

一个用 [MyGo](https://github.com/egoist/mygo) 原生 UI 写的桌面番茄钟，纯 Go、无 cgo、无 webview——窗口由 MyGo 直接在 GPU 上绘制，打包后是一个约 9 MB 的单二进制应用。

## 功能

- **圆形进度环倒计时**：自绘 `ui.Path` + `Path.Trim`，弧线随剩余时间消退；专注为番茄红，休息为绿色
- **专注 / 休息自动切换**：专注结束自动进入休息并计数，反之亦然
- **4 种内置铃声**（经典提示 / 清脆铃声 / 三连钟 / 木鱼）：全部由代码实时合成（`audio.NewSound` + 衰减正弦泛音），不打包任何音频文件；支持音量调节与试听
- **桌面通知**：会话切换时发送系统通知（可在设置中开关）
- **系统托盘**：菜单栏番茄图标，运行时图标旁显示倒计时；关闭窗口后驻留托盘继续计时
- **设置持久化**：铃声、音量、通知开关保存在应用数据目录（`settings.json`），窗口大小位置也会被记住

## 快速开始

需要 Go 1.27.3+（`GOTOOLCHAIN=auto` 时会自动下载）。

```sh
go tool mygo dev     # 开发模式，热重载
go test              # 无窗口测试视图（mygo 的 ui.Tester 模拟点击）
go tool mygo build   # 打包 .app / DMG，输出在 build/
```

## 实现要点

- 视图是「状态 → 界面」的声明式函数，每帧重建（类似 Dear ImGui / egui）
- 倒计时基于墙钟（`endsAt`），配合 `c.Now()` + `c.After(time.Second)` 请求下一帧；窗口被遮挡也不丢时间
- 自定义绘制用 `Element.Draw(func(p *ui.Painter, r ui.Rect))`：进度环、番茄计数图标都是这个 API
- 托盘图标与 App 图标都是程序生成的 PNG（见 `tools/genicon`）

## 项目结构

```
main.go          # 应用状态 + 视图 + 铃声合成 + 托盘/通知
main_test.go     # ui.Tester 无窗口测试
tools/genicon/   # 生成 resources/icon.png 的图标脚本
mygo.json        # 应用名、标识符、版本、输出目录
resources/       # 应用图标
```

## 许可

MIT
