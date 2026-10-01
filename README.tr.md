# ReAppKit

[English](README.md)

ReAppKit, Windows bilgisayarı uygulamalar ve kişisel ayarlar için iki terminal seçim ekranıyla hazırlamayı amaçlar. İlk MVP geliştiriliyor; henüz yayımlanmış bir sürüm yok.

## Mevcut MVP

- Windows 11 x64 hedeflenir. Derlenen çalıştırılabilir dosyanın hedef bilgisayarda Go kurulumuna ihtiyacı yoktur.
- Katalogda 20 uygulama bulunur: Chrome, VS Code, Git, Obsidian, WezTerm, WhatsApp, Brave, ChatGPT, Sublime Text, Notion, Spotify, MarkText, WinRAR, Okular, DBeaver Community, PowerToys, Espanso, LibreOffice, Thunderbird ve TranslucentTB. [Katalog referansı](catalog/README.md) paket kimliklerini, kaynakları ve doğrulanan kapsamları içerir. Farklı bir JSON uygulama kataloğu verilebilir.
- Uzun uygulama listelerinde PgUp/PgDn, Home/End ve fare tekerleği kullanılabilir. Uzun özet ve sonuç ekranları yön tuşları, PgUp/PgDn veya fare tekerleğiyle kaydırılabilir.
- Program varsayılan olarak **demo modunda** açılır. Seçim, özet ve sonuç akışını gösterir; bilgisayarı değiştirmez.
- `--apply`, ancak kullanıcı seçim yapıp özet ekranında ayrıca onay verince gerçek işlemleri açar. Git, Sublime Text, WinRAR, LibreOffice ve Thunderbird machine kapsamında başlangıç UAC işçisinde kurulur; WezTerm aynı işçide auto kapsamını kullanır. Diğer hazır uygulamalar kullanıcı sürecinde çalışır. Kurulu olanlar atlanır; bir işin hatası bağımsız işleri durdurmaz.
- İsteğe bağlı ayar kataloğu, kullanıcının verdiği yapılandırma dosyalarını kopyalar. Hedef dosya varsa değiştirmeden önce yanında yedek oluşturur. Kişisel kaynak ve hedef yolları bilinmediği için hazır ayar sunulmaz.
- Makine tarafından okunabilir sonuçlar varsayılan olarak `%LOCALAPPDATA%\ReAppKit\runs` konumuna yazılır. `--results-dir` ile değiştirilebilir.
- Özet onayından sonra UAC veya dosya değişikliği başlamadan WinGet/Windows desteği ve seçilen uygulamaların kurulu durumu kontrol edilir. Eksik uygulama varsa ilgili WinGet CDN veya Microsoft Store sunucusuna süre sınırlı HTTPS kontrolü yapılır. Kontrol başarısızsa işlem açıklamayla durur; yeniden onaylayarak tekrar denenebilir. Yalnız yerel dosya işi seçilmişse, ayar bir uygulamayı gerektirmedikçe WinGet veya internet aranmaz.

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

WhatsApp ve ChatGPT `msstore`, diğer hazır uygulamalar `winget` kaynağını kullanır. Katalogda `source: "winget"` (varsayılan) veya `source: "msstore"` seçilebilir. Store uygulamaları kullanıcı sürecinde kalır ve kapsam bayrağı gönderilmez. Kaynak koşulları henüz incelenmediyse ön kontrol hatası, PowerShell'de koşulları incelemek için çalıştırılacak tam komutu gösterir. ReAppKit kaynak veya paket koşullarını otomatik kabul etmez. Yeni uygulamalar ve gerçek Store kurulumları temiz Windows VM testi gerektirir.

Bağlantı kontrolü seçilen hazır kaynakların sunucusunu sınar; bütün kurucu indirme adreslerini veya özel WinGet kaynaklarını doğrulamaz. Kurulum başarısını garanti etmez. Ek HTTPS kontrolü atlandığında bile kurulu uygulama sorgusu kaynak erişimine ihtiyaç duyabilir.

Chrome, VS Code ve Obsidian kullanıcı kapsamında, Git tüm kullanıcılar için machine kapsamında kurulur. Katalogdaki `scope` alanı `user`, `machine` veya `auto` olabilir; boş alan `user` olur. `auto` kurucu seçimini WinGet’e bırakır, kullanıcı kapsamı garantisi vermez ve yönetici uyarısı gösterir. Kurulum başlamadan tek Windows UAC onayı alınır; normal uygulamalar kullanıcı sürecinde, yönetici işleri geçici yönetici işçisinde sırayla çalışır. Profiller, yeniden başlatma sonrası devam ve Windows ses/kayıt defteri ayarları henüz yoktur. Bazı üçüncü taraf yükleyiciler yine de yönetici izni isteyebilir; devam etmek istemiyorsan bu istemi iptal et. Uygulama kurulumu için WinGet gerekir. Ayrı lisans kabulü isteyen paketler etkileşimsiz modda başarısız olabilir; ReAppKit paket anlaşmalarını otomatik kabul etmez. Sonuç ekranındaki `r`, özete dönüp yeniden onaylama olanağı verir; tamamlanmış işler kontrol edilip atlanır. Onay öncesi kurulu durumu gösterme henüz yoktur. Yerel dosya kopyalama akışı doğrulandı. Windows VM'de Chrome, VS Code, Git ve Obsidian kurulum işleri başarı bildirdi; tekrar çalıştırma ve temiz Windows kabulü açık kaldı. Windows Uygulama Denetimi ilk yerel derlemeyi engelledi, sonraki imzasız derleme geliştirme bilgisayarında açıldı; bu nedenle çalıştırma davranışı her cihazda doğrulanmalıdır.


## WezTerm kapsam düzeltmesi — 30 Eylül 2026

WezTerm `scope: "auto"` kullanır; kurulum komutuna `--scope` gönderilmez ve yönetici uyarısı korunur. Önceki VM denemesinde zorlanan kapsamla `0x8a150010` görüldü. Düzeltmenin gerçek kurulum ve UAC davranışı VM'de henüz doğrulanmadı; ayrı yönetici aşaması uygulanmış değildir. Önceki doğrulama kaydındaki dört uygulamalı katalog bu değişiklikten önceki durumu anlatır.

VM PowerShell'de önce `winget install --id wez.wezterm --exact --source winget` ile kapsam belirtmeden dene. Ardından yeni derlemede yalnız WezTerm'i seçerek kurulumu ve tekrar çalıştırmada `skipped` sonucunu doğrula. İlk komut da başarısız olursa `winget --info`, `winget show --id wez.wezterm --exact --source winget` çıktısını ve hatanın işaret ettiği WinGet günlüğünü incele.

Yerel doğrulama: `go vet ./...` ve derleme başarılı. Ana program, katalog, core, settings ve UI testleri geçti. WinGet test çalıştırılabilirini Windows Uygulama Denetimi engelledi; bu paketin testleri derlendi ancak çalıştırılamadı.


## Ayrı yönetici onayı — 30 Eylül 2026

Git kurucusu kullanıcı kapsamında da UAC isteyebilir. Git ve WezTerm yönetici aşamasında işaretlenir. Özet onayından sonra terminal arayüzü duraklatılır; önce normal uygulamalar, sonra yönetici işleri ve son olarak dosya ayarları çalışır. Kurulu olmayan her yönetici uygulamasından hemen önce `e` ve Enter ile devam, `h` ile vazgeçme onayı alınır. Windows UAC penceresi çıkınca ayrıca onaylanmalıdır; görünmüyorsa görev çubuğunu veya Alt+Tab'ı kontrol et. Kurulu uygulamalar bu onaydan önce kontrol edilip atlanır. Vazgeçilen işin bağımlı ayarları engellenir; bağımsız işler devam eder. Kurucu iptali okunabilir izin iptali açıklaması verir; kendiliğinden yeniden denenmez. UAC otomatik kabul edilmez, uygulamanın tamamı yönetici olarak çalıştırılmaz. Git'in kullanıcı kapsamı ve WezTerm'in kapsamsız komutu korunur.

Önceki açıklamalardaki “ayrı yönetici aşaması uygulanmadı” sınırı bu sürümde ayrı kullanıcı onayı bakımından güncellenmiştir; programatik UAC yükseltmesi uygulanmış değildir. Temiz VM'de toplu kurulum ve gerçek izin penceresi görünürlüğü henüz doğrulanmadı.


## Başlangıçta tek UAC onayı — 30 Eylül 2026

Bu akış önceki uygulama başına e/h sorularının yerini alır. Özet onayından sonra kurulu yönetici uygulamaları kontrol edilir. Eksik iş varsa bilgisayarda değişiklik başlamadan bir kez Windows yönetici onayı alınır ve geçici yönetici işçisi başlatılır. Normal uygulamalar ilk kullanıcı sürecinde kalır; yönetici işleri ayrı süreçte sırayla çalışır. Git artık tüm kullanıcılar için machine kapsamında kurulur; WezTerm auto kapsamını korur. İşçi yalnız seçilmiş machine/auto yönetici paketlerini kabul eder ve oturum sonunda kapanır. Kalıcı servis veya UAC ilkesi değişikliği yapılmaz. İlk UAC iptal edilirse kurulum ve dosya değişikliği başlamaz. Bütün yönetici uygulamaları zaten kuruluysa yükseltme istenmez. Ek lisans veya etkileşimli kurucu adımı isteyen başka paketler başarısız olabilir; gelecekteki her paketin gözetimsiz çalışması garanti değildir. Gerçek yükseltme ve temiz VM denemesi kabul testidir.
