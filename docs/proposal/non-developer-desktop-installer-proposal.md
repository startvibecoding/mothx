# Self-Extracting Desktop Installer Proposal

> Date: 2026-07-14
> Status: Proposal

## 背景

当前 release 产物主要面向开发者：

- Windows: `zip` + `install.bat`
- macOS: `tar.gz` + `install.sh`
- npm / PyPI: 需要 Node.js、Python、pipx 或 npm 环境

原先考虑过 Windows `.exe` installer 和 macOS `.pkg` / `.dmg`，但这会把正式出包绑定到 Windows/macOS runner、平台签名工具、macOS notarization 和额外证书链路。现在约束调整为：**所有面向 Windows x86_64、macOS x86_64、macOS arm64 的非开发人员安装包都必须能在 Linux 上生成**。

在这个约束下，推荐改为自解压脚本安装器：

- Windows: 一个 `.bat` 文件，内嵌压缩后的 `mothx.exe` payload。
- macOS: 一个 `.sh` 文件，内嵌压缩后的 `mothx` payload。

这类安装包类似一些驱动或离线 installer：用户下载单个脚本，双击或在终端运行，脚本把尾部 payload 解压到本机安装目录，并完成 PATH、快捷方式、卸载脚本等配置。

本文中的 `x86` 按现有 Go 构建目标理解为 `amd64` / `x86_64`。不建议把 32 位 Windows x86 作为首批目标；当前 `Makefile` 已说明 32 位构建受 larksuite SDK 依赖限制。

## 目标

- 在 Linux CI 上生成全部首批非开发人员安装包。
- 覆盖三类目标：
  - Windows x86_64
  - macOS x86_64
  - macOS arm64
- 产物是单文件，用户不需要手动解压 zip/tarball。
- 安装后 `mothx` 在默认终端中可直接使用。
- 支持升级覆盖安装和卸载。
- 保留当前 `zip` / `tar.gz` / npm / PyPI 作为开发者和自动化安装渠道。

## 非目标

- 不在首期提供 Windows `.exe` / `.msi` 原生安装器。
- 不在首期提供 macOS `.pkg` / `.dmg` 原生安装器。
- 不在首期解决 SmartScreen / Gatekeeper 的全部信任问题。
- 不做后台自动更新服务。
- 不把 CLI 包装成完整桌面 GUI 应用。
- 不改变 `settings.json`、session、provider、sandbox 等运行时行为。
- 不移除现有 release 产物。

## 推荐方案

采用“Linux 出包 + 自解压脚本”的方案：

| 用户平台 | Go 构建目标 | 用户下载产物 | 安装方式 |
| --- | --- | --- | --- |
| Windows x86_64 | `GOOS=windows GOARCH=amd64` | `mothx-<version>-windows-x64-install.bat` | 双击或 PowerShell/cmd 运行 |
| macOS x86_64 | `GOOS=darwin GOARCH=amd64` | `mothx-<version>-macos-x64-install.sh` | Terminal 中运行 |
| macOS arm64 | `GOOS=darwin GOARCH=arm64` | `mothx-<version>-macos-arm64-install.sh` | Terminal 中运行 |

macOS 首期建议分成 x64 和 arm64 两个 `.sh`。这样 Linux 侧不需要额外引入 Mach-O universal binary 工具链，也避免一个脚本内嵌两个二进制导致体积翻倍。

下载页可以仍然只展示一个“Download for macOS”入口，页面或下载脚本按 User-Agent 提示 Apple Silicon / Intel；GitHub Release 中保留两个明确命名的安装文件。

## 产物命名

建议 release 中新增：

```text
dist/installers/mothx-<version>-windows-x64-install.bat
dist/installers/mothx-<version>-macos-x64-install.sh
dist/installers/mothx-<version>-macos-arm64-install.sh
dist/installers/installer-checksums.txt
```

现有产物继续保留：

```text
dist/zip/mothx-<version>-windows-amd64.zip
dist/tarball/mothx-<version>-darwin-amd64.tar.gz
dist/tarball/mothx-<version>-darwin-arm64.tar.gz
```

## Payload 格式

每个 installer 文件由两部分组成：

```text
script header
payload marker
base64(compressed archive)
```

Windows `.bat` 使用 base64 编码后的 zip payload：

```text
mothx.exe
README.txt
uninstall.bat
```

macOS `.sh` 使用 base64 编码后的 `tar.gz` payload：

```text
mothx
README.txt
uninstall-mothx.sh
```

不直接拼接二进制流，原因是：

- `.bat` 对二进制尾部兼容性差，文本编码、换行和下载工具都可能破坏 payload。
- base64 文本更容易在 GitHub Release、浏览器下载、杀毒扫描和 shell 处理中保持稳定。
- checksums 可以同时覆盖完整 installer 文件和内部二进制。

## Windows `.bat` 安装器设计

### 安装位置

默认当前用户安装：

```text
%LOCALAPPDATA%\Programs\MothX\
%LOCALAPPDATA%\Programs\MothX\bin\mothx.exe
```

理由：

- 不需要管理员权限。
- 不污染 `C:\Program Files`。
- 升级和卸载对普通用户更稳定。

### 安装动作

`.bat` 执行以下步骤：

1. 检查当前系统架构，非 x86_64 时给出错误提示。
2. 创建临时目录。
3. 从自身文件中定位 payload marker。
4. 将 marker 后的 base64 内容写入临时 `.b64`。
5. 用 PowerShell 解码为 zip。
6. 用 PowerShell `Expand-Archive` 解压 zip。
7. 复制 `mothx.exe` 到 `%LOCALAPPDATA%\Programs\MothX\bin\`。
8. 将安装目录加入用户 PATH。
9. 写入 `uninstall.bat`。
10. 创建桌面快捷方式 `MothX Serve`，双击后打开 PowerShell 并运行 `mothx serve`。
11. 执行 `mothx --version` 做安装后验证。

Windows 10/11 默认带 PowerShell 5+，因此可以把 PowerShell 作为解码和解压依赖。若 PowerShell 不存在，脚本给出明确错误，不再尝试复杂 fallback。

### PATH 处理

不要直接用 `setx PATH "%PATH%;..."`，因为 `%PATH%` 可能包含系统 PATH，且有长度截断风险。建议在 `.bat` 内调用 PowerShell 修改用户级 Environment：

```powershell
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $installBin) {
  [Environment]::SetEnvironmentVariable("Path", "$userPath;$installBin", "User")
}
```

脚本需要提示用户重新打开终端。

### 卸载

安装器写入：

```text
%LOCALAPPDATA%\Programs\MothX\uninstall.bat
```

卸载动作：

- 删除 `%LOCALAPPDATA%\Programs\MothX\bin\mothx.exe`。
- 从用户 PATH 中移除安装目录。
- 删除桌面快捷方式 `MothX Serve`。
- 删除 `%LOCALAPPDATA%\Programs\MothX`。
- 默认保留用户数据：

```text
%APPDATA%\mothx\
%USERPROFILE%\.mothx\
```

可以支持安装器自身参数：

```bat
mothx-<version>-windows-x64-install.bat --uninstall
mothx-<version>-windows-x64-install.bat --target "%LOCALAPPDATA%\Programs\MothX"
mothx-<version>-windows-x64-install.bat --quiet
```

### 桌面快捷方式

创建桌面快捷方式 `MothX Serve.lnk`，目标固定为 PowerShell，并使用安装目录中的绝对路径启动 serve 模式：

```text
powershell.exe -NoExit -ExecutionPolicy Bypass -Command "& '<install>\bin\mothx.exe' serve"
```

这里不依赖 PATH，因此用户刚安装完成、不重新打开终端，也可以直接通过桌面快捷方式启动 `mothx serve`。

## macOS `.sh` 安装器设计

### 安装位置

首期建议默认安装到用户目录，避免 `sudo`：

```text
~/.local/bin/mothx
~/.local/share/mothx/
```

理由：

- `.sh` 不是 signed/notarized `.pkg`，要求用户再输入管理员密码会降低信任。
- 非开发用户安装失败率更低。
- 不会修改 `/usr/local/bin` 权限和 ownership。

安装器需要检查 `~/.local/bin` 是否在 PATH 中。如果不在，尝试更新 shell rc 文件。

### shell rc 更新

安装器识别当前 shell：

```text
zsh  -> ~/.zshrc
bash -> ~/.bash_profile，缺失时创建 ~/.bashrc
fish -> ~/.config/fish/config.fish
```

追加带 marker 的 PATH block，便于后续卸载：

```sh
# >>> mothx path >>>
export PATH="$HOME/.local/bin:$PATH"
# <<< mothx path <<<
```

如果无法安全识别 shell rc 文件，只打印下一步提示，不强行修改未知文件。

### 安装动作

`.sh` 执行以下步骤：

1. 检查 `uname -s` 必须是 `Darwin`。
2. 检查 `uname -m` 和安装器架构匹配。
3. 创建临时目录。
4. 从自身文件中定位 payload marker。
5. 解码 base64 payload 为 `tar.gz`。
6. 解压 payload。
7. 复制 `mothx` 到 `~/.local/bin/mothx`。
8. `chmod +x ~/.local/bin/mothx`。
9. 写入 `~/.local/share/mothx/uninstall-mothx.sh`。
10. 必要时更新 shell rc 的 PATH block。
11. 执行 `mothx --version` 做安装后验证。

### base64 兼容性

macOS 和 GNU coreutils 的 `base64` 参数不完全一致。为避免安装脚本依赖差异，建议使用 Python fallback：

1. 优先 `base64 -d`。
2. 失败时尝试 `base64 -D`。
3. 再失败时尝试 `/usr/bin/python3 -c` 解码。

如果三者都不可用，给出明确错误。

### 卸载

安装器写入：

```text
~/.local/share/mothx/uninstall-mothx.sh
```

卸载动作：

- 删除 `~/.local/bin/mothx`。
- 删除 shell rc 中的 `mothx path` block。
- 删除 `~/.local/share/mothx`。
- 默认保留用户数据：

```text
~/.mothx/
```

可以支持安装器自身参数：

```sh
sh mothx-<version>-macos-arm64-install.sh --uninstall
sh mothx-<version>-macos-arm64-install.sh --prefix "$HOME/.local"
sh mothx-<version>-macos-arm64-install.sh --quiet
```

## Linux 出包流程

所有产物都在 Linux runner 中生成：

```text
build binaries
  windows/amd64 -> bin/mothx-windows-amd64.exe
  darwin/amd64  -> bin/mothx-darwin-amd64
  darwin/arm64  -> bin/mothx-darwin-arm64

build payload archives
  windows zip   -> build/installers/payload/windows-x64.zip
  macos tar.gz  -> build/installers/payload/macos-x64.tar.gz
  macos tar.gz  -> build/installers/payload/macos-arm64.tar.gz

render script templates
  packaging/windows/self-extract/install.bat.tmpl
  packaging/macos/self-extract/install.sh.tmpl

append base64 payload
  script header + marker + base64 archive

verify installer file checksums
```

## 构建目录建议

新增目录：

```text
packaging/
  windows/
    self-extract/
      install.bat.tmpl
      uninstall.bat.tmpl
      README.txt
  macos/
    self-extract/
      install.sh.tmpl
      uninstall-mothx.sh.tmpl
      README.txt
scripts/
  build-self-extract-installers.sh
```

`scripts/build-self-extract-installers.sh` 负责：

- 校验 `bin/` 中目标二进制存在。
- 生成 payload archive。
- 渲染模板中的版本号、架构、目标路径默认值。
- 追加 base64 payload。
- 生成 `dist/installers/installer-checksums.txt`。
- 可选做一次 payload round-trip 验证。

## Makefile 入口

建议新增：

```makefile
.PHONY: dist-installers

dist-installers: build-windows build-darwin
	./scripts/build-self-extract-installers.sh $(VERSION)
```

总 release 可以变成：

```makefile
dist: dist-linux dist-darwin dist-windows dist-installers checksums
```

也可以先不并入 `dist`，只在 release workflow 中显式执行 `make dist-installers`，降低首期风险。

## GitHub Actions 设计

现有 `.github/workflows/release.yml` 已运行在 `ubuntu-latest`。在同一个 job 中追加：

```text
make ui-install
make ui-build
make dist
make dist-installers
```

上传 release artifacts 时增加：

```text
dist/installers/*.bat
dist/installers/*.sh
dist/installers/installer-checksums.txt
```

不再需要 Windows runner 或 macOS runner。后续如果要加签名安装器，可以新增独立 workflow，不影响 self-extract installer。

## 安装后体验

### Windows

用户路径：

1. 下载 `mothx-<version>-windows-x64-install.bat`。
2. 双击运行，或右键选择“Run as administrator”但默认不需要。
3. 安装完成后打开新的 PowerShell。
4. 输入：

```powershell
mothx
```

### macOS

macOS 对从浏览器下载的 shell 脚本通常不会像 `.pkg` 那样提供顺滑双击安装体验。推荐下载页直接给出一行命令：

```bash
sh ~/Downloads/mothx-<version>-macos-arm64-install.sh
```

用户路径：

1. 下载对应架构的 `.sh`。
2. 打开 Terminal。
3. 运行安装命令。
4. 重新打开 Terminal。
5. 输入：

```bash
mothx
```

## 安全和信任说明

自解压脚本满足 Linux-only 出包，但安全体验弱于 signed native installer。

### Windows

- `.bat` 下载后可能被浏览器、Defender 或 SmartScreen 提示风险。
- `.bat` 无法提供和签名 `.exe` 同等级的 publisher 体验。
- 可以对内嵌的 `mothx.exe` 后续用 Linux Authenticode 工具签名，但 `.bat` 本身仍是脚本。

### macOS

- `.sh` 不走 `.pkg` notarization 体验。
- 用户需要在 Terminal 中运行脚本。
- Gatekeeper 主要拦截 GUI app / pkg / dmg，CLI 脚本体验取决于下载来源、浏览器 quarantine 属性和用户执行方式。

### 缓解措施

- Release 页面提供 SHA256 checksum。
- 安装器启动时打印版本、目标平台、安装路径和 payload checksum。
- 安装前显示将要执行的文件写入和 PATH 修改。
- 默认安装到用户目录，避免管理员权限。
- 提供 `--dry-run` 显示安装动作。
- 保留 zip/tarball，方便安全敏感用户手动安装。

## 版本和升级策略

- 安装器版本来自 git tag，和当前 `VERSION` / `LDFLAGS` 保持一致。
- 覆盖安装即升级：新二进制替换旧二进制。
- 安装器写入 `~/.local/share/mothx/install.json` 或 Windows 对应目录中的 `install.json`，记录：

```json
{
  "version": "v0.0.0",
  "platform": "darwin-arm64",
  "installDir": "...",
  "installedAt": "..."
}
```

- 用户配置和 sessions 不随升级删除。
- 首期不做自动更新；下载页、release note、`mothx doctor` 或后续 update checker 可以提示新版本。

## 下载页展示策略

GitHub Release 和文档下载页给普通用户展示：

```text
Download for Windows
Download for macOS Apple Silicon
Download for macOS Intel
```

高级下载折叠区再展示：

- Windows zip
- macOS Intel tar.gz
- macOS Apple Silicon tar.gz
- Linux tar.gz / deb
- npm
- PyPI

需要明确提示：

- Windows `.bat` 是单文件安装器。
- macOS `.sh` 需要在 Terminal 中运行。
- 所有安装器默认安装到用户目录，不需要管理员权限。

## 验收标准

### Windows x86_64

- 在干净 Windows 11 x64 VM 中双击 `.bat` 安装成功。
- 安装过程默认不需要管理员权限。
- 新开的 PowerShell 中 `mothx --version` 可执行。
- 用户 PATH 包含 `%LOCALAPPDATA%\Programs\MothX\bin`。
- 桌面快捷方式 `MothX Serve` 可打开 PowerShell 并运行 `mothx serve`。
- `uninstall.bat` 可以删除程序文件、PATH 项和快捷方式。
- 卸载后用户数据默认保留。

### macOS x86_64

- 在 Intel macOS 机器或 VM 上运行 `.sh` 安装成功。
- `~/.local/bin/mothx --version` 可执行。
- 新开的 Terminal 中 `mothx --version` 可执行。
- shell rc 中的 PATH block 可重复运行且不重复追加。
- `uninstall-mothx.sh` 可以删除程序文件和 PATH block。
- 卸载后用户数据默认保留。

### macOS arm64

- 在 Apple Silicon macOS 机器上运行 `.sh` 安装成功。
- `file ~/.local/bin/mothx` 显示 arm64 Mach-O。
- 新开的 Terminal 中 `mothx --version` 可执行。
- shell rc 中的 PATH block 可重复运行且不重复追加。
- `uninstall-mothx.sh` 可以删除程序文件和 PATH block。
- 卸载后用户数据默认保留。

### Linux CI

- `make dist-installers` 在 `ubuntu-latest` 上完成。
- 不依赖 Windows runner。
- 不依赖 macOS runner。
- 不依赖 `codesign`、`pkgbuild`、`productbuild`、`hdiutil`、`notarytool`。
- 生成的 installer 文件可 round-trip 解码 payload。
- `installer-checksums.txt` 包含全部 installer 文件。

## 风险和取舍

### 非开发用户对脚本的信任

`.bat` / `.sh` 比 native installer 更容易被用户认为“不安全”。但它换来了 Linux-only 出包、实现简单、可审计、跨平台行为一致。下载页需要清楚解释安装路径和卸载方式。

### macOS 不是双击安装

`.sh` 在 macOS 上不适合作为双击 GUI installer。首期接受这个取舍，因为目标从“最原生体验”调整为“Linux-only 生成的单文件安装包”。如果未来要提升 macOS 非开发体验，应另行提供 signed/notarized `.pkg` 作为增强产物。

### PATH 修改容易出错

shell rc 和 Windows user PATH 都可能有历史内容。脚本必须做到：

- 幂等。
- 可卸载。
- 不覆盖用户原有内容。
- 修改前后输出明确提示。

### 单文件体积

base64 会让 payload 体积增加约三分之一。MothX 是单 Go binary，这个成本可以接受。若体积变大，可以改用更高压缩率的 payload，但安装端必须避免引入额外解压依赖。

## 分阶段计划

### Phase 1: 生成自解压安装器

- 新增 Windows `.bat` 模板和 macOS `.sh` 模板。
- 新增 uninstall 模板。
- 新增 `scripts/build-self-extract-installers.sh`。
- 生成三个 installer 文件。
- 在 Linux CI 中做 payload round-trip 验证。

### Phase 2: 手工 VM 验证

- Windows 11 x64 VM 验证双击安装、PATH、快捷方式、卸载。
- macOS Intel 验证 `.sh` 安装、PATH block、卸载。
- macOS Apple Silicon 验证 `.sh` 安装、PATH block、卸载。
- 根据真实终端行为修正提示文案。

### Phase 3: Release 集成

- 将 `make dist-installers` 加入 release workflow。
- 上传 `dist/installers/*.bat` 和 `dist/installers/*.sh`。
- 更新 checksums。
- 更新下载页和 getting started 文档。

### Phase 4: 增强分发

- 继续保留 self-extract installer 作为 Linux-only 基线产物。
- 可选新增 signed Windows `.exe` installer。
- 可选新增 signed/notarized macOS `.pkg` / `.dmg`。
- 可选新增 winget / Homebrew cask，指向增强产物或现有 tarball。
