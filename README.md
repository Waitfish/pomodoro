# 番茄钟 (Pomodoro)

一个用 [MyGo](https://github.com/egoist/mygo) 原生 UI 写的桌面番茄钟，纯 Go、无 cgo、无 webview——窗口由 MyGo 直接在 GPU 上绘制，打包后是一个约 9 MB 的单二进制应用。

![浅色模式](docs/screenshot.png#gh-light-mode-only)
![深色模式](docs/screenshot-dark.png#gh-dark-mode-only)

> 截图由 `ui.Tester` 无窗口渲染直接导出（`SHOTS=1 go test -run TestScreenshots`），左：浅色专注模式；右：深色休息模式 + 设置面板。

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

- 视图是「状态 → 界面」的声明式函数，窗口每次绘制时重建（类似 Dear ImGui / egui）
- **计时在后台 goroutine**：每 200ms 经 `Window.Update` 投递主线程。视图只在窗口绘制时运行，最小化/遮挡/关到托盘就没有帧——若靠 `c.After` 驱动，时钟会冻结、到期不提醒（1.0.3 修复的正是这个）。倒计时基于墙钟（`endsAt`），迟到或丢失的帧不丢时间
- 自定义绘制用 `Element.Draw(func(p *ui.Painter, r ui.Rect))`：进度环、番茄计数图标都是这个 API
- 托盘图标与 App 图标都是程序生成的 PNG（见 `tools/genicon`）

## 安装

### Ubuntu（PPA）

```sh
sudo add-apt-repository ppa:daiwj2024/ppa
sudo apt update
sudo apt install fangqiezhong
```

安装后可从应用菜单启动，或运行 `fangqiezhong` 命令。PPA 由 Launchpad 构建签名，覆盖 amd64 与 arm64，支持 Ubuntu 22.04（Jammy）及之后所有受支持的版本（24.04 Noble、26.04 Resolute 等）。20.04（Focal）已 EOL，Launchpad 不再为它发布构建，可直接用 Releases 的 `.deb`。

> PPA 的 apt 下载域名为 `ppa.launchpadcontent.net`（新版 Launchpad），实测中国大陆可直连。若个别网络不可达，可配置代理或直接从 Releases 下载 `.deb` 安装。

### 其他平台

从 [Releases](../../releases) 下载对应平台的 `.dmg` / `-setup-*.exe` / `.deb`，或用 `install.sh` / `install.ps1` 一键安装。

## 发布（GitHub Actions）

推送 `v*` 标签即自动跨平台构建并发布 Release：

```sh
git tag v1.0.1 && git push origin v1.0.1
```

`.github/workflows/release.yml` 会并行跑三个矩阵（产物文件名用 `mygo.json` 里的 `slug: fangqiezhong`，mygo 对「茄」的默认读音是 jié，需显式指定）：

| 平台 | Runner | 产物 |
| --- | --- | --- |
| macOS | macos-latest | `.app` + `.dmg`（arm64 & Intel） |
| Windows | ubuntu-latest（纯 Go 交叉编译） | NSIS 安装包 `-setup-*.exe`（amd64 & arm64） |
| Linux | ubuntu-latest | `.deb`、`.tar.gz`、`install.sh`（amd64 & arm64） |

所有产物连同**签名过的更新清单**（`update-*.json`，供应用自动更新）上传到草稿 Release，全部构建成功后自动发布。更新签名私钥保存在仓库 Secret `MYGO_UPDATER_PRIVATE_KEY` 中，公钥在 `mygo.json`；发布说明取自 `CHANGELOG.md` 对应版本小节。

> 未配置代码签名证书：macOS 产物为 ad-hoc 签名（其他 Mac 首次打开需右键“打开”），Windows 产物未签名（SmartScreen 会警告）。

推标签同时触发 `ppa.yml`：交叉构建 Linux 二进制 → 组装 `debian/` 源码包 → `dput` 上传 Launchpad PPA（Launchpad 构建机无网络且 Go 版本过旧，故源码包内携带预构建二进制，rules 只做安装）。上传凭据为仓库 Secret `LAUNCHPAD_SSH_PRIVATE_KEY`。

## 项目结构

```
main.go             # 应用状态 + 视图 + 铃声合成 + 托盘/通知
main_test.go        # ui.Tester 无窗口测试
screenshot_test.go  # 渲染 README 截图（SHOTS=1 go test -run TestScreenshots）
debian/             # PPA 源码包打包（native 格式，安装预构建二进制）
tools/genicon/      # 生成 resources/icon.png 的图标脚本
mygo.json           # 应用名、slug、标识符、版本、更新与 Linux 打包配置
resources/          # 应用图标
docs/               # README 截图
```

## 许可

MIT
