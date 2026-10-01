# ReAppKit

[English](README.md)

Terminal arayüzünde uygulamaları ve kişisel yapılandırma dosyalarını seçerek Windows bilgisayarını hazırla.

**Durum:** İlk geliştirme aşamasında. Henüz yayımlanmış sürüm yok; tüm kataloğun temiz Windows sistemlerinde kurulumu doğrulanmaya devam ediyor.

## Özellikler

- Uygulamaları ve ayarları iki ekranda seç, çalıştırmadan önce özeti incele.
- 20 uygulamalık katalogdan WinGet ve Microsoft Store üzerinden kurulum yap.
- Değişiklik başlamadan gereksinimleri ve kurulu uygulamaları kontrol et.
- Seçilen uygulamalar gerektiriyorsa başlangıçta tek Windows yönetici onayı al.
- Kurulu uygulamaları atla; bir iş başarısız olduğunda bağımsız işlere devam et.
- Mevcut yapılandırma dosyalarını değiştirmeden önce yedekle.
- Her çalıştırma için JSON sonuç raporu oluştur.

## Gereksinimler

- Windows 11 x64.
- Uygulama kurulumu için Windows App Installer ile sağlanan WinGet.
- Seçilen paket kaynaklarına ve kurucu indirme adreslerine erişim.
- **Yalnız kaynak koddan derlemek için** Go 1.27 veya üzeri. Derlenen uygulamanın hedef bilgisayarda Go kurulumuna ihtiyacı yoktur.

## Hızlı başlangıç

Depo kökünde derleyip demoyu aç:

```powershell
go build -o ReAppKit.exe .
.\ReAppKit.exe
```

Demo modu, uygulama kurmadan veya dosya değiştirmeden seçim ekranlarını denemeni sağlar.

Seçilen işleri uygulamak için:

```powershell
.\ReAppKit.exe --apply
```

Uygulamaları seç, ayarları seç ve özet ekranını onayla. ReAppKit başlamadan önce planı kontrol eder. Yönetici onayı gerekiyorsa herhangi bir kurulum veya dosya değişikliğinden önce istenir. Bu onayın iptal edilmesi çalıştırmayı durdurur.

Uygulamalar sırayla kurulur, ardından dosya ayarları uygulanır. Yönetici izni gerektirmeyen uygulamalar kullanıcı sürecinde kalır.

## Uygulama kataloğu

| Kategori | Uygulamalar |
| --- | --- |
| Tarayıcılar | Chrome, Brave |
| Geliştirme | Visual Studio Code, Git, WezTerm, Sublime Text, DBeaver Community |
| Üretkenlik | Obsidian, Notion, ChatGPT, MarkText, Okular, LibreOffice |
| İletişim | WhatsApp, Thunderbird |
| Medya | Spotify |
| Araçlar | WinRAR, PowerToys, Espanso, TranslucentTB |

WhatsApp, ChatGPT ve Okular Microsoft Store, diğer uygulamalar topluluk WinGet kaynağını kullanır. Paket kimlikleri, kurulum kapsamları ve kaynak bağlantıları için [katalog referansına](catalog/README.md) bak.

## Kişisel yapılandırma dosyaları

Kopyalanacak dosyaları tanımlayan bir `settings.json` oluştur:

```json
[
  {
    "id": "wezterm-config",
    "title": "WezTerm yapılandırması",
    "description": "Terminal yapılandırmamı kopyala",
    "category": "Terminal",
    "source": "C:\\Users\\You\\Setup\\wezterm.lua",
    "target": "C:\\Users\\You\\.wezterm.lua",
    "requires_app": "wez.wezterm"
  }
]
```

Örnek yolları kendi mutlak dosya yollarınla değiştir ve çalıştır:

```powershell
.\ReAppKit.exe --apply --settings-catalog settings.json
```

Mevcut hedef dosya değiştirilmeden önce yedeklenir. İsteğe bağlı `requires_app` alanı, ayarı ilgili uygulamanın kurulumuna bağlar. Uygulama aynı planda seçilmediyse zaten kurulu olmalıdır. Uygulama bağımlılığı olmayan yerel dosya ayarları WinGet veya internet gerektirmez.

## Kontroller

| İşlem | Kontrol |
| --- | --- |
| Gezin veya bir öğeyi seç | ↑/↓, Space/Enter veya fare tıklaması |
| Uzun listede gezin | PgUp/PgDn, Home/End veya fare tekerleği |
| Devam et / geri dön | Tab veya → / ← veya Backspace |
| Özeti onayla | Enter |
| Özeti veya sonuçları kaydır | ↑/↓, PgUp/PgDn veya fare tekerleği |
| Sonuçlardan yeniden dene | `r`, ardından özeti tekrar onayla |
| Çık | `q` veya Ctrl+C |

## Komut satırı seçenekleri

| Seçenek | Amaç |
| --- | --- |
| `--apply` | Özet onayından sonra gerçek işlemleri aç |
| `--apps-catalog <yol>` | Özel JSON uygulama kataloğu yükle |
| `--settings-catalog <yol>` | Kişisel dosya ayarlarını yükle |
| `--results-dir <yol>` | Sonuç raporu klasörünü belirle |

Sonuç raporları varsayılan olarak `%LOCALAPPDATA%\ReAppKit\runs` konumuna yazılır.

## Mevcut sınırlamalar

Kaynak ve paket koşulları otomatik kabul edilmez. Ön kontrolde kaynak koşulları eksikse ReAppKit, yeniden denemeden önce çalıştırılacak PowerShell komutunu gösterir. Kurulum sırasında paket koşulları için WinGet aynı terminalde açık onayınızı ister; reddederseniz o uygulama kurulmaz. Bazı kurucular ek etkileşim isteyebilir; bağlantı kontrolü her indirmenin başarılı olacağını garanti etmez.

Uygulama hesabına giriş, ücretli lisanslar ve kişisel uygulama ayarları kurulumdan ayrı tamamlanır. Hazır Windows görünüm ve ses ayarları, tekrar kullanılabilir profiller ve yeniden başlatma sonrası devam henüz sunulmaz.
