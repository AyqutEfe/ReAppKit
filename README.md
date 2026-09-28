# ReAppKit

[Türkçe](README.tr.md)

ReAppKit helps set up a Windows PC through two terminal selection screens: applications and personal settings. An early MVP is now in development; there is no published release yet.

## Current MVP

- Targets Windows 11 x64. The built executable does not require Go on the target PC.
- Includes a starter WinGet catalog for Chrome, VS Code, Git, and Obsidian. WezTerm's WinGet installer does not support user scope, so it is deferred to a separate, explicitly approved admin phase. You can supply a different JSON app catalog.
- Opens in **demo mode by default**. Demo mode shows the selection, summary, and results flow without changing the computer.
- `--apply` enables real work only after the user selects items and confirms on the summary screen. Application installs request user scope. Existing installations are skipped, and independent jobs continue when one fails.
- An optional settings catalog can copy user-provided configuration files. Existing target files receive a sibling backup before replacement. No settings are bundled because source files and destinations are personal.
- Machine-readable run results are written to `%LOCALAPPDATA%\ReAppKit\runs` unless `--results-dir` is set.

## Build and try

Install Go 1.27 or later to build from source, then run in this directory:

```powershell
go build -o ReAppKit.exe .
.\ReAppKit.exe
```

The command above is a safe demo. To allow selected work after a second confirmation in the UI:

```powershell
.\ReAppKit.exe --apply
```

To provide personal file settings, create a JSON array like this and pass `--settings-catalog settings.json`:

```json
[
  {
    "id": "wezterm-config",
    "title": "WezTerm configuration",
    "description": "Copy my existing configuration",
    "category": "Terminal",
    "source": "C:\\Users\\You\\Setup\\wezterm.lua",
    "target": "C:\\Users\\You\\.wezterm.lua",
    "requires_app": "wez.wezterm"
  }
]
```

Replace both paths with absolute paths on your PC. `requires_app` is optional. A setting depending on an app waits for that app when both are selected; otherwise the app must already be installed.

## Current limits

Only user-scope app installs and explicit file-copy settings are wired into the UI. Admin-required steps, profiles, restart continuation, and Windows sound or registry settings are not implemented. Some third-party installers may still request elevation; cancel that prompt if you do not want to proceed. WinGet must be available for application installs. Packages requiring separate license acceptance may fail in non-interactive mode; ReAppKit does not automatically accept package agreements. The results screen offers `r` to return to the summary and confirm another run; completed jobs are checked and skipped. Installed-state preview before confirmation is still pending. A local file-copy run was verified. In a Windows VM, Chrome, VS Code, Git, and Obsidian installation jobs reported success; a repeat run and clean-machine acceptance remain open. Windows App Control blocked the first local build but allowed a later unsigned build on the development PC, so execution must be checked per device.
