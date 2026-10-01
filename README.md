# ReAppKit

[Türkçe](README.tr.md)

ReAppKit helps set up a Windows PC through two terminal selection screens: applications and personal settings. An early MVP is now in development; there is no published release yet.

## Current MVP

- Targets Windows 11 x64. The built executable does not require Go on the target PC.
- Includes 20 apps: Chrome, VS Code, Git, Obsidian, WezTerm, WhatsApp, Brave, ChatGPT, Sublime Text, Notion, Spotify, MarkText, WinRAR, Okular, DBeaver Community, PowerToys, Espanso, LibreOffice, Thunderbird and TranslucentTB. The [catalog reference](catalog/README.md) records package IDs, sources and verified scopes. You can supply a different JSON app catalog.
- Large application lists support PgUp/PgDn, Home/End and the mouse wheel. Long summary and result screens can be scrolled with arrow keys, PgUp/PgDn or the mouse wheel.
- Opens in **demo mode by default**. Demo mode shows the selection, summary, and results flow without changing the computer.
- `--apply` enables real work only after the user selects items and confirms on the summary screen. Git, Sublime Text, WinRAR, LibreOffice and Thunderbird request machine scope in the initial UAC worker; WezTerm uses automatic scope there. Other bundled apps run in the original user process. Existing installations are skipped, and independent jobs continue when one fails.
- An optional settings catalog can copy user-provided configuration files. Existing target files receive a sibling backup before replacement. No settings are bundled because source files and destinations are personal.
- Machine-readable run results are written to `%LOCALAPPDATA%\ReAppKit\runs` unless `--results-dir` is set.
- After summary confirmation, preflight checks WinGet/Windows support and selected app installation states before UAC or file changes. Missing apps trigger a bounded HTTPS check to their WinGet CDN or Microsoft Store endpoint. A failed check stops the run with guidance; confirm again to retry. Local file-only runs do not require WinGet or internet unless a setting requires an installed app.

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

WhatsApp and ChatGPT use `msstore`; other bundled entries use `winget`. Catalog entries accept `source: "winget"` (the default) or `source: "msstore"`. Store apps stay in the user process and do not send a scope flag. If source agreements have not been reviewed, the preflight error shows the exact PowerShell command to review them before retrying. ReAppKit does not automatically accept source or package agreements. New entries and real Store installs still require clean Windows VM verification.

The connectivity probe checks the selected built-in source endpoint, not every vendor's download server or a custom WinGet source. It does not guarantee installation success. Installed-app detection can itself require source access even when the extra HTTPS probe is skipped.

App catalogs support `user`, `machine`, and `auto` scope; omitted scope defaults to `user`. Automatic scope lets WinGet select the installer and requires the admin warning, since it does not guarantee a user-scope install. One initial Windows UAC approval authorizes a temporary worker; ordinary apps stay in the original user process and admin apps run sequentially in the worker. Profiles, restart continuation, and Windows sound or registry settings are not implemented. Some third-party installers may still request elevation; cancel that prompt if you do not want to proceed. WinGet must be available for application installs. Packages requiring separate license acceptance may fail in non-interactive mode; ReAppKit does not automatically accept package agreements. The results screen offers `r` to return to the summary and confirm another run; completed jobs are checked and skipped. Installed-state preview before confirmation is still pending. A local file-copy run was verified. In a Windows VM, Chrome, VS Code, Git, and Obsidian installation jobs reported success; a repeat run and clean-machine acceptance remain open. Windows App Control blocked the first local build but allowed a later unsigned build on the development PC, so execution must be checked per device.


## WezTerm scope fix — 2026-09-30

WezTerm now uses `scope: "auto"`; the install command omits `--scope` while retaining the admin warning. The previous VM error was `0x8a150010` with forced scope. Installation and UAC behavior still require VM verification; a separate elevation phase is not implemented.

In VM PowerShell, diagnose first with `winget install --id wez.wezterm --exact --source winget`. Then test the corrected executable with only WezTerm selected, and repeat to verify `skipped`. If the scope-free command fails, capture `winget --info`, `winget show --id wez.wezterm --exact --source winget`, and the WinGet log referenced by the failure.


## Interactive admin phase — 2026-09-30

Git's installer can request elevation even with user scope. Git and WezTerm are marked for the admin phase. After summary confirmation, the terminal UI pauses while ordinary apps run first, followed by admin apps and finally file settings. Immediately before each uninstalled admin app, type `e` and Enter to proceed or `h` to decline. Confirm the Windows UAC dialog when it appears; check the taskbar/Alt+Tab if needed. Installed apps are checked and skipped before this prompt. Declining blocks dependent settings but independent jobs continue. Installer cancellation gets a UAC hint and is not automatically retried. The app does not auto-accept UAC or elevate the entire process. `--scope user` for Git and scope-free WezTerm are unchanged.

Native UAC visibility and a fresh VM batch install still require verification.


## One initial UAC approval — 2026-09-30

This supersedes the per-app e/h prompts above. After summary confirmation, installed admin apps are checked first. If any remain, ReAppKit obtains one Windows UAC approval before changing the computer and starts a temporary elevated worker. Ordinary apps remain in the original user process; admin apps run sequentially in the worker. Git now uses machine scope (all users), while WezTerm keeps auto scope. The worker only accepts the selected machine/auto admin packages and exits when the run ends. No persistent service or UAC-policy change is made. Cancelling initial UAC aborts the run before installs or file changes. With no pending admin apps, no elevation is requested. A vendor requiring additional license or interactive setup steps can still fail; arbitrary future packages are not guaranteed to be unattended. Native elevation and a fresh VM run remain acceptance tests.
