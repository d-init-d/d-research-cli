# d-research-cli

Windows-first CLI cho workflow nghiên cứu sâu với TUI và headless mode.

## Yêu cầu

- Go 1.22+
- Python 3.10+
- Node 20+
- Windows 10/11 (v0.1)

## Cài đặt dev

```powershell
go build -o d-research.exe ./cmd/d-research
```

## Lệnh chính

```text
d-research
d-research run --prompt "..." --headless --events jsonl
d-research auth login --provider openrouter
d-research model configure --provider openrouter --model google/gemini-2.5-flash --global
d-research search providers
d-research browser doctor
d-research kb status
d-research config show
d-research doctor
```

## Workspace

```text
./kb/
./research-output/
./simulation/
./.d-research/
```

Global config: `%APPDATA%\d-research\config.json`  
Secrets: Windows Credential Manager (`d-research-cli`) hoặc `D_RESEARCH_*` ENV.

## Kiểm thử

```powershell
go test ./...
```