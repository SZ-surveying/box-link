# box-link

[English](../README.md) | [中文](README_zh.md)

`box-link` 是一个面向 macOS 的直连组网工具，用来管理电脑与 box 设备之间的直接网络连接。

它提供一个 Go 二进制，同时支持两种入口：

- 命令行模式，适合熟悉终端的用户
- 本地网页模式，适合更习惯点按钮的用户

项目把接口探测、日志、诊断、打包和发布流程都收敛在同一套代码里。

更多架构细节见 `design.md`，执行计划见 `todo.md`。

## 前置要求

- macOS
- 本地开发时需要安装 Go
- `on`、`off`，以及多数情况下的 `ui` 需要 `sudo` 才能修改网卡设置

## 命令

随时可以先看帮助：

```bash
box-link --help
```

各命令说明：

- `info`：显示当前生效的配置文件路径和最终配置
- `iface`：解析当前应该用于 box 直连的网卡接口
- `status`：输出接口、路由和 ping 状态
- `doctor`：输出一份简明诊断结果
- `on`：给目标接口配置主机 IP，并测试 box 是否可达
- `off`：把目标接口恢复为 DHCP
- `ui`：启动本地网页控制台
- `version`：输出版本号

## 配置

`box-link` 现在默认使用这个配置文件：

```text
~/.box-link/config.toml
```

如果文件不存在，首次运行时会自动创建。

默认样板如下：

```toml
# box-link configuration
config_path = "/Users/<you>/.box-link/config.toml"
iface = ""
host_ip = "192.168.10.3"
box_ip = "192.168.10.2"
netmask = "255.255.255.0"
hardware_port_pattern = "AX88179A"
log_level = "DEBUG"
listen_addr = "127.0.0.1:18888"
```

说明：

- `config_path` 会写入配置文件中，同时也会显示在 `box-link info` 和 Web UI 里
- `iface = ""` 对应 UI 里的 `Iface override = (auto)`
- 如果你想手动指定网卡，可以改成 `iface = "en7"` 这种形式

环境变量仍然支持，并且会覆盖配置文件中的值：

```bash
BOX_CONFIG_PATH
BOX_IFACE
BOX_HOST_IP
BOX_IP
BOX_NETMASK
BOX_HARDWARE_PORT_PATTERN
BOX_LOG_LEVEL
BOX_LISTEN_ADDR
```

接口解析优先级：

1. 如果显式设置了 `iface` / `BOX_IFACE` 且接口存在，直接使用
2. 用 `networksetup -listallhardwareports` 匹配 `hardware_port_pattern` / `BOX_HARDWARE_PORT_PATTERN`
3. 查找已经绑定 `host_ip` / `BOX_HOST_IP` 的 `en*` 接口
4. 选择优先级最高的活跃外接 `en*`
5. 最后回退到编号最高的外接 `en*`

## 常见使用方式

命令行流程：

```bash
box-link info
box-link iface
sudo box-link on
box-link status
sudo box-link off
```

网页模式：

```bash
sudo box-link ui
```

然后打开：

```text
http://127.0.0.1:18888
```

Web UI 的 Config 面板会显示：

- config file path
- iface override
- host IP
- box IP
- netmask
- hardware match pattern
- log level
- listen address

## 开发

本项目使用 `just` 统一开发命令：

```bash
just fmt
just build
just test
just run status
just package
just package-release
just package-pkg
just homebrew-formula
just homebrew-tap
```

说明：

- `just package`：构建当前主机平台的压缩包
- `just package-release`：构建 `darwin/arm64` 和 `darwin/amd64` 两个发布包
- `just package-pkg`：在 macOS 上生成 `.pkg` 安装包
- `just homebrew-formula`：根据发布产物生成 Homebrew formula
- `just homebrew-tap`：直接写入 tap 仓库的 `Formula/` 目录
- `./scripts/check.sh` 支持快捷别名：`--pre-commit`、`--ci`、`--full`

## 打包

本地打包：

```bash
just package
```

发布目标打包：

```bash
just package-release
```

构建 `.pkg` 安装包：

```bash
just package-pkg
```

生成 Homebrew formula：

```bash
just homebrew-formula rainy/box-link
```

直接写入 tap 仓库：

```bash
just homebrew-tap rainy/box-link ../homebrew-tools
```

产物默认输出到 `dist/`：

- `box-link-<version>-<goos>-<goarch>.tar.gz`
- `box-link-<version>.pkg`
- `checksums.txt`

每个压缩包中包含：

- `box-link`
- `install.sh`
- `README.md`

解压后可直接执行：

```bash
./install.sh
```

## 发布流程

Tag 格式：

```text
v<major>.<minor>.<patch>
```

本地辅助命令：

```bash
just release 0.1.0
```

GitHub Actions 会自动：

- 校验 tag 格式
- 在 Ubuntu 上构建 tarball
- 生成并附带 Homebrew formula
- 在 macOS runner 上构建并附带 `.pkg`
- 用 `git-cliff` 生成 release notes
- 把所有产物发布到同一个 GitHub Release

## 面向非开发用户的分发方式

### Homebrew Tap

适合技术用户，一条命令安装：

```bash
brew tap <org>/tools
brew install box-link
```

建议方式：

- 单独创建一个 tap 仓库，比如 `homebrew-tools`
- 使用 GitHub Releases 中的发布压缩包
- 通过 `just homebrew-formula <owner/repo>` 生成可用的 formula
- 或通过 `just homebrew-tap <owner/repo> <tap-dir>` 直接写入 tap 仓库

### macOS Installer Package

适合不熟悉命令行、希望双击安装的用户。

建议方式：

- 保持 `just package-release` 负责二进制打包
- 用 `just package-pkg` 生成 `.pkg`
- 正式分发前再做签名和 notarization

例如：

```bash
just package-pkg com.example.box-link
just homebrew-formula rainy/box-link
```
