# ReAppKit

[Türkçe](README.tr.md)

Set up a Windows PC by selecting applications and personal configuration files in a terminal interface.

**Status:** Early development. No published release yet; full-catalog installation on clean Windows systems is still being validated.

## Features

- Select applications and settings in two screens, then review the plan before running it.
- Install from a catalog of 20 applications through WinGet and Microsoft Store.
- Check prerequisites and installed applications before making changes.
- Request one initial Windows administrator approval when selected applications need it.
- Skip installed applications and continue independent jobs if another job fails.
- Back up existing configuration files before replacing them.
- Save a JSON result report for each run.

## Requirements

- Windows 11 x64.
- WinGet, provided by Windows App Installer, for application installation.
- Access to the selected package sources and installer downloads.
- Go 1.27 or later **only when building from source**. The resulting executable does not require Go on the target PC.

## Quick start

Build from the repository root and open the demo:

```powershell
go build -o ReAppKit.exe .
.\ReAppKit.exe
```

Demo mode lets you try the selection screens without installing applications or changing files.

To run selected work:

```powershell
.\ReAppKit.exe --apply
```

Select applications, select settings, then confirm the summary. ReAppKit checks the plan before starting. If an administrator approval is required, it is requested before any installation or file change. Cancelling that approval stops the run.

Applications run in sequence, followed by file settings. Applications that do not need administrator privileges remain in the user process.

## Application catalog

| Category | Applications |
| --- | --- |
| Browsers | Chrome, Brave |
| Development | Visual Studio Code, Git, WezTerm, Sublime Text, DBeaver Community |
| Productivity | Obsidian, Notion, ChatGPT, MarkText, Okular, LibreOffice |
| Communication | WhatsApp, Thunderbird |
| Media | Spotify |
| Utilities | WinRAR, PowerToys, Espanso, TranslucentTB |

WhatsApp, ChatGPT and Okular use Microsoft Store; the other entries use the community WinGet source. See the [catalog reference](catalog/README.md) for package identifiers, installation scopes and source references.

## Personal configuration files

Create a `settings.json` file describing the files to copy:

```json
[
  {
    "id": "wezterm-config",
    "title": "WezTerm configuration",
    "description": "Copy my terminal configuration",
    "category": "Terminal",
    "source": "C:\\Users\\You\\Setup\\wezterm.lua",
    "target": "C:\\Users\\You\\.wezterm.lua",
    "requires_app": "wez.wezterm"
  }
]
```

Replace the example paths with your own absolute paths, then run:

```powershell
.\ReAppKit.exe --apply --settings-catalog settings.json
```

Existing target files receive a backup before replacement. The optional `requires_app` field makes the setting depend on that application's installation. If the application is not selected in the same plan, it must already be installed. Local file settings without an application dependency do not require WinGet or internet access.

## Controls

| Action | Control |
| --- | --- |
| Navigate or select an item | ↑/↓, Space/Enter, or mouse click |
| Navigate a long list | PgUp/PgDn, Home/End, or mouse wheel |
| Continue / go back | Tab or → / ← or Backspace |
| Confirm the summary | Enter |
| Scroll the summary or results | ↑/↓, PgUp/PgDn, or mouse wheel |
| Retry from the results | `r`, then confirm the summary again |
| Exit | `q` or Ctrl+C |

## Command-line options

| Option | Purpose |
| --- | --- |
| `--apply` | Enable real operations after summary confirmation |
| `--apps-catalog <path>` | Load a custom JSON application catalog |
| `--settings-catalog <path>` | Load personal file settings |
| `--results-dir <path>` | Choose the result report directory |

Result reports default to `%LOCALAPPDATA%\ReAppKit\runs`.

## Current limitations

Source and package agreements are not automatically accepted. If a source needs agreement review during preflight, ReAppKit shows the PowerShell command to run before retrying. During installation, pending package agreements are shown by WinGet in the same terminal for your explicit response; declining leaves that application uninstalled. Some installers can require additional interaction, and a connectivity check does not guarantee every download will succeed.

Application sign-in, paid licenses and personal app setup remain separate from installation. Built-in Windows appearance and sound settings, reusable profiles and continuation after a restart are not available yet.
