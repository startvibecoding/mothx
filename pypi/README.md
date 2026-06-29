# VibeCoding

A terminal-based AI coding assistant packaged for PyPI. The wheel includes the native `vibecoding` binary for your platform and exposes the `vibecoding` command.

## Installation

```bash
pipx install vibecoding-installer
```

If you do not use pipx:

```bash
python -m pip install --user vibecoding-installer
```

## Usage

```bash
vibecoding
vibecoding -P "Write a hello world in Go"
vibecoding --provider deepseek-openai --model deepseek-v4-flash
```

## Supported PyPI Wheels

- Linux x86_64, arm64, ppc64le, s390x
- Linux musl x86_64, arm64
- macOS x86_64, arm64
- Windows x64, arm64

For other platforms, use the npm package or the one-line install scripts from the main repository.

## Uninstall

```bash
pipx uninstall vibecoding-installer
# or
python -m pip uninstall vibecoding-installer
```

## Links

- GitHub: <https://github.com/startvibecoding/vibecoding>
- Documentation: <https://github.com/startvibecoding/vibecoding/tree/main/docs>
- License: MIT
