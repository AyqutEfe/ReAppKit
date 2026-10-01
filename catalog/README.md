# Application catalog

Checked on 2026-10-01 for Windows 11 x64. This verifies package identifiers and manifest scope, not installation on a clean machine. Versions below are evidence snapshots; installs are not pinned.

| App | Package ID | Source | Scope | Evidence |
| --- | --- | --- | --- | --- |
| WhatsApp | `9NKSQGP7F2NH` | msstore | user | [Official Windows distribution](https://www.whatsapp.com/download/windows) |
| Brave | `Brave.Brave` | winget | user | [154.1.96.59](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/b/Brave/Brave/154.1.96.59/Brave.Brave.installer.yaml) |
| ChatGPT | `9NT1R1C2HH7J` | msstore | user | [Official Windows distribution](https://help.openai.com/en/articles/9982051) |
| Sublime Text | `SublimeHQ.SublimeText.4` | winget | machine | [4.0.0.421500](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/s/SublimeHQ/SublimeText/4/4.0.0.421500/SublimeHQ.SublimeText.4.installer.yaml) |
| Notion | `Notion.Notion` | winget | user | [7.36.1](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/n/Notion/Notion/7.36.1/Notion.Notion.installer.yaml) |
| Spotify | `Spotify.Spotify` | winget | user | [1.3.1.234.g59d6bf59](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/s/Spotify/Spotify/1.3.1.234.g59d6bf59/Spotify.Spotify.installer.yaml) |
| MarkText | `MarkText.MarkText` | winget | user | [0.19.1](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/m/MarkText/MarkText/0.19.1/MarkText.MarkText.installer.yaml) |
| WinRAR | `RARLab.WinRAR` | winget | machine | [7.23.0](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/r/RARLab/WinRAR/7.23.0/RARLab.WinRAR.installer.yaml) |
| Okular | `KDE.Okular` | winget | user | [26.08.1](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/k/KDE/Okular/26.08.1/KDE.Okular.installer.yaml) |
| DBeaver Community | `DBeaver.DBeaver.Community` | winget | user | [26.2.0](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/d/DBeaver/DBeaver/Community/26.2.0/DBeaver.DBeaver.Community.installer.yaml) |
| PowerToys | `Microsoft.PowerToys` | winget | user | [0.101.2362.0](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/m/Microsoft/PowerToys/0.101.2362.0/Microsoft.PowerToys.installer.yaml) |
| Espanso | `Espanso.Espanso` | winget | user | [2.4.1](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/e/Espanso/Espanso/2.4.1/Espanso.Espanso.installer.yaml) |
| LibreOffice | `TheDocumentFoundation.LibreOffice` | winget | machine | [26.8.0.3](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/t/TheDocumentFoundation/LibreOffice/26.8.0.3/TheDocumentFoundation.LibreOffice.installer.yaml) |
| Thunderbird | `Mozilla.Thunderbird` | winget | machine | [157.0](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/m/Mozilla/Thunderbird/157.0/Mozilla.Thunderbird.installer.yaml) |
| TranslucentTB | `CharlesMilette.TranslucentTB` | winget | user | [2026.1](https://raw.githubusercontent.com/microsoft/winget-pkgs/master/manifests/c/CharlesMilette/TranslucentTB/2026.1/CharlesMilette.TranslucentTB.installer.yaml) |

Microsoft Store apps run in the original user process, with no `--scope` argument sent to WinGet. Source or package agreements are not automatically accepted. If a Store source agreement is pending, run the exact `winget list --id <ID> --exact --source msstore` command shown by ReAppKit, review its terms, then retry. Store availability can depend on account, region and organizational policy.

TranslucentTB is an MSIX package in the community WinGet source and installs for the user. Its MSIX framework dependencies remain managed by WinGet. Sublime Text, WinRAR, LibreOffice and Thunderbird join the initial UAC worker with machine scope. Spotify stays in the non-elevated user process (its manifest prohibits elevation).

Application sign-in, paid licenses, personal preferences and Windows appearance changes require their own application setup; selecting an app installs it only.
