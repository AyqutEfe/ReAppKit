# ReAppKit

[English](README.md)

ReAppKit, Windows bilgisayarı uygulamalar ve kişisel ayarlar için iki terminal seçim ekranıyla hazırlamayı amaçlar. İlk MVP geliştiriliyor; henüz yayımlanmış bir sürüm yok.

## Mevcut MVP

- Windows 11 x64 hedeflenir. Derlenen çalıştırılabilir dosyanın hedef bilgisayarda Go kurulumuna ihtiyacı yoktur.
- Başlangıç WinGet kataloğunda Chrome, VS Code, Git ve Obsidian bulunur. WezTerm'in WinGet kurucusu kullanıcı kapsamını desteklemediği için ayrı ve açık onaylı yönetici aşamasına ertelendi. Farklı bir JSON uygulama kataloğu verilebilir.
- Program varsayılan olarak **demo modunda** açılır. Seçim, özet ve sonuç akışını gösterir; bilgisayarı değiştirmez.
- `--apply`, ancak kullanıcı seçim yapıp özet ekranında ayrıca onay verince gerçek işlemleri açar. Uygulamalar kullanıcı kapsamında kurulmaya çalışılır. Kurulu olanlar atlanır; bir işin hatası bağımsız işleri durdurmaz.
- İsteğe bağlı ayar kataloğu, kullanıcının verdiği yapılandırma dosyalarını kopyalar. Hedef dosya varsa değiştirmeden önce yanında yedek oluşturur. Kişisel kaynak ve hedef yolları bilinmediği için hazır ayar sunulmaz.
- Makine tarafından okunabilir sonuçlar varsayılan olarak `%LOCALAPPDATA%\ReAppKit\runs` konumuna yazılır. `--results-dir` ile değiştirilebilir.

## Derleme ve deneme

Kaynak koddan derlemek için Go 1.27 veya yenisini kurup bu klasörde çalıştır:

```powershell
go build -o ReAppKit.exe .
.\ReAppKit.exe
```

Bu komut güvenli demo modunu açar. Seçilen işleri arayüzde ikinci kez onayladıktan sonra uygulamaya izin vermek için:

```powershell
.\ReAppKit.exe --apply
```

Kendi ayar dosyalarını eklemek için aşağıdaki biçimde bir JSON dizisi oluşturup `--settings-catalog settings.json` parametresini ver:

```json
[
  {
    "id": "wezterm-config",
    "title": "WezTerm yapılandırması",
    "description": "Mevcut yapılandırmamı kopyala",
    "category": "Terminal",
    "source": "C:\\Users\\You\\Setup\\wezterm.lua",
    "target": "C:\\Users\\You\\.wezterm.lua",
    "requires_app": "wez.wezterm"
  }
]
```

Kaynak ve hedefi kendi bilgisayarındaki mutlak yollarla değiştir. `requires_app` isteğe bağlıdır. Gerekli uygulama aynı planda seçildiyse ayar onu bekler; seçilmediyse uygulamanın zaten kurulu olması gerekir.

## Şimdiki sınırlar

Arayüze yalnız kullanıcı kapsamındaki uygulama kurulumları ve açıkça tanımlanmış dosya kopyalama ayarları bağlıdır. Yönetici izni isteyen ayrı aşama, profiller, yeniden başlatma sonrası devam ve Windows ses/kayıt defteri ayarları henüz yoktur. Bazı üçüncü taraf yükleyiciler yine de yönetici izni isteyebilir; devam etmek istemiyorsan bu istemi iptal et. Uygulama kurulumu için WinGet gerekir. Ayrı lisans kabulü isteyen paketler etkileşimsiz modda başarısız olabilir; ReAppKit paket anlaşmalarını otomatik kabul etmez. Sonuç ekranındaki `r`, özete dönüp yeniden onaylama olanağı verir; tamamlanmış işler kontrol edilip atlanır. Onay öncesi kurulu durumu gösterme henüz yoktur. Yerel dosya kopyalama akışı doğrulandı. Windows VM'de Chrome, VS Code, Git ve Obsidian kurulum işleri başarı bildirdi; tekrar çalıştırma ve temiz Windows kabulü açık kaldı. Windows Uygulama Denetimi ilk yerel derlemeyi engelledi, sonraki imzasız derleme geliştirme bilgisayarında açıldı; bu nedenle çalıştırma davranışı her cihazda doğrulanmalıdır.
