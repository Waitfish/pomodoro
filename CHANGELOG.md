# Changelog

## 1.0.2 - 2026-11-02

- Linux 打包元数据完善：`webview: false`（纯 native UI，去掉 WebKitGTK 依赖）、托盘依赖 `libayatana-appindicator3-1`、`/usr/bin/fangqiezhong` 命令、desktop entry 修复
- 新增 Launchpad PPA 发布通道：`debian/` 源码包 + `ppa.yml` 工作流，Ubuntu 用户可通过 `ppa:daiwj2024/fangqiezhong` 安装

## 1.0.1 - 2026-11-02

- 修正包名拼音：slug 显式设为 `fangqiezhong`（mygo 对「茄」默认取了 jié 读音，此前生成 `fan-jia-zhong`）
- README 增加应用截图

## 1.0.0 - 2026-11-02

- 首个版本：圆形进度环番茄钟，专注/休息自动切换并计数
- 4 种内置合成铃声（经典提示 / 清脆铃声 / 三连钟 / 木鱼），支持音量与试听
- 桌面通知（可开关）；系统托盘驻留，菜单栏显示倒计时
- 设置持久化到应用数据目录
