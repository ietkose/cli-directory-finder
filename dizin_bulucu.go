package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Terminal renk kodları (ANSI)
const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorRed    = "\033[31m"
	ColorCyan   = "\033[36m"
	ColorBold   = "\033[1m"
	ColorGray   = "\033[90m"
)

// Okunabilir dosya boyutu formatı
func formatBoyut(bytes int64) string {
	const unit = 1024.0
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	div := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	exp := 0
	for div >= unit && exp < len(units)-1 {
		div /= unit
		exp++
	}
	return fmt.Sprintf("%.2f %s", div, units[exp])
}

// Dosya içinde metin arama (maksimum 10MB boyuta kadar olan dosyaları okur)
func dosyaIcerikAra(path string, aranan string, maxBoyut int64) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxBoyut {
		return false
	}

	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Uzun satırlara karşı buffer kapasitesini ayarlıyoruz
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	arananKucuk := strings.ToLower(aranan)
	for scanner.Scan() {
		if strings.Contains(strings.ToLower(scanner.Text()), arananKucuk) {
			return true
		}
	}
	return false
}

func main() {
	// Custom Terminal Yardım Menüsü (Help Page)
	flag.Usage = func() {
		fmt.Printf("\n%s📖 Dizin ve Dosya Bulucu — Terminal Yardım Menüsü%s\n", ColorCyan+ColorBold, ColorReset)
		fmt.Println(strings.Repeat("=", 65))
		fmt.Printf("%sKullanım:%s\n", ColorYellow, ColorReset)
		fmt.Println("  go run dizin_bulucu.go [PARAMETRELER]")
		fmt.Println("  .\\dizin_bulucu.exe [PARAMETRELER]")
		fmt.Println("\n" + ColorYellow + "Parametreler ve Seçenekler:" + ColorReset)
		fmt.Printf("  %-25s %s\n", "-path <dizin>", "Arama kök dizini (Varsayılan: '.')")
		fmt.Printf("  %-25s %s\n", "-search <metin>", "Aranacak dosya veya dizin adı")
		fmt.Printf("  %-25s %s\n", "-content <metin>", "Dosya İÇERİĞİNDE aranacak metin (Maks 10MB)")
		fmt.Printf("  %-25s %s\n", "-ext <uzantı>", "Dosya uzantı filtresi (Örn: .pdf, go, png)")
		fmt.Printf("  %-25s %s\n", "-only-dir", "Sadece klasörleri/dizinleri listeler")
		fmt.Printf("  %-25s %s\n", "-only-file", "Sadece dosyaları listeler")
		fmt.Printf("  %-25s %s\n", "-depth <sayı>", "Maksimum dizin arama derinliği (-1: sınırsız)")
		fmt.Printf("  %-25s %s\n", "-no-hidden", "Gizli dosya ve klasörleri atlar ('.' ile başlayanlar)")
		fmt.Printf("  %-25s %s\n", "-exclude <klasörler>", "Taranmayacak klasör isimleri (Örn: node_modules,.git)")
		fmt.Printf("  %-25s %s\n", "-days <gün>", "Son N gün içinde değiştirilmiş dosyaları filtreler")
		fmt.Printf("  %-25s %s\n", "-output <dosya>", "Tarama sonuçlarını belirtilen dosyaya kaydeder")
		fmt.Printf("  %-25s %s\n", "-h, -help", "Bu yardım ekranını görüntüler")
		fmt.Println(strings.Repeat("-", 65))
		fmt.Printf("%sÖrnek Kullanımlar:%s\n", ColorGreen, ColorReset)
		fmt.Println("  1. UZANTI ARAMASI   : .\\dizin_bulucu.exe -ext pdf")
		fmt.Println("  2. İSMEN ARAMA      : .\\dizin_bulucu.exe -search rapor -depth 2")
		fmt.Println("  3. İÇERİK ARAMASI   : .\\dizin_bulucu.exe -content \"TODO\" -ext .go")
		fmt.Println("  4. GELİŞMİŞ TARAMA  : .\\dizin_bulucu.exe -days 7 -exclude \"node_modules,.git\" -output sonuc.txt")
		fmt.Println(strings.Repeat("=", 65) + "\n")
	}

	// Terminal Parametreleri
	aramaYolu := flag.String("path", ".", "Arama yapılacak kök dizin")
	arananMetin := flag.String("search", "", "Aranacak kelime veya dosya/dizin adı")
	icerikAra := flag.String("content", "", "Dosya İÇERİĞİNDE aranacak metin")
	sadeceDizin := flag.Bool("only-dir", false, "Sadece dizinleri/klasörleri getir")
	sadeceDosya := flag.Bool("only-file", false, "Sadece dosyaları getir")
	uzanti := flag.String("ext", "", "Sadece belirli bir uzantıyı ara (Örn: .go, pdf, png)")
	maxDerinlik := flag.Int("depth", -1, "Maksimum dizin derinliği (-1: sınırsız)")
	gizliHaric := flag.Bool("no-hidden", false, "Gizli dosya ve klasörleri atla ('.' ile başlayanlar)")
	haricKlasorler := flag.String("exclude", "", "Taranmayacak klasörler (virgülle ayırın, Örn: node_modules,.git,vendor)")
	gunSiniri := flag.Int("days", 0, "Son N gün içinde değiştirilmiş dosyaları filtrele (0: devre dışı)")
	ciktiDosyasi := flag.String("output", "", "Sonuçların kaydedileceği dosya (Örn: sonuclar.txt)")

	flag.Parse()

	// Parametre Kontrolleri
	if *sadeceDizin && *sadeceDosya {
		fmt.Printf("%s[HATA]%s -only-dir ve -only-file parametreleri aynı anda kullanılamaz!\n", ColorRed, ColorReset)
		return
	}

	if *arananMetin == "" && *uzanti == "" && *icerikAra == "" && *gunSiniri == 0 {
		flag.Usage()
		return
	}

	// Kök yolu temizle (WalkDir karşılaştırmaları için)
	kokYol := filepath.Clean(*aramaYolu)
	kokDerinlik := strings.Count(kokYol, string(os.PathSeparator))

	// Performans için dize ön işleme (Döngü dışında 1 kez yapılıyor)
	arananKucuk := strings.ToLower(*arananMetin)
	uzantiHedef := strings.ToLower(*uzanti)
	if uzantiHedef != "" && !strings.HasPrefix(uzantiHedef, ".") {
		uzantiHedef = "." + uzantiHedef
	}

	// Hariç tutulacak klasör listesi
	var exclList []string
	if *haricKlasorler != "" {
		for _, item := range strings.Split(*haricKlasorler, ",") {
			trimmed := strings.TrimSpace(strings.ToLower(item))
			if trimmed != "" {
				exclList = append(exclList, trimmed)
			}
		}
	}

	// Çıktı dosyasını hazırlama
	var outputFile *os.File
	var err error
	if *ciktiDosyasi != "" {
		outputFile, err = os.Create(*ciktiDosyasi)
		if err != nil {
			fmt.Printf("%s[HATA]%s Çıktı dosyası oluşturulamadı: %v\n", ColorRed, ColorReset, err)
			return
		}
		defer outputFile.Close()
	}

	logVeYaz := func(msg string) {
		fmt.Println(msg)
		if outputFile != nil {
			// Renk kodlarını dosyaya yazarken temizle
			cleanMsg := msg
			for _, color := range []string{ColorReset, ColorBlue, ColorGreen, ColorYellow, ColorRed, ColorCyan, ColorBold, ColorGray} {
				cleanMsg = strings.ReplaceAll(cleanMsg, color, "")
			}
			outputFile.WriteString(cleanMsg + "\n")
		}
	}

	logVeYaz(fmt.Sprintf("\n%s🕰️ Taramaya Başlanıyor...%s", ColorCyan, ColorReset))
	logVeYaz(fmt.Sprintf("🦉 Kök Dizin : %s", kokYol))
	if *arananMetin != "" {
		logVeYaz(fmt.Sprintf("🦉 Aranan İsim: %s", *arananMetin))
	}
	if *icerikAra != "" {
		logVeYaz(fmt.Sprintf("🦉 İçerik Araması: %s", *icerikAra))
	}
	if uzantiHedef != "" {
		logVeYaz(fmt.Sprintf("🦉 Uzantı     : %s", uzantiHedef))
	}
	if *maxDerinlik >= 0 {
		logVeYaz(fmt.Sprintf("🦉 Maks. Derinlik: %d", *maxDerinlik))
	}
	if *gunSiniri > 0 {
		logVeYaz(fmt.Sprintf("🦉 Tarih Sınırı : Son %d gün", *gunSiniri))
	}
	logVeYaz(strings.Repeat("-", 60))

	baslangic := time.Now()
	toplamTarama := 0
	bulunanSayisi := 0
	var toplamBoyut int64 = 0

	err = filepath.WalkDir(kokYol, func(yol string, d fs.DirEntry, err error) error {
		if err != nil {
			logVeYaz(fmt.Sprintf("%s⚠️ Erişim Engellendi: %s%s", ColorRed, yol, ColorReset))
			return nil
		}

		temizYol := filepath.Clean(yol)

		// 1. Ana dizinin kendisini geç
		if temizYol == kokYol {
			return nil
		}

		toplamTarama++
		isim := d.Name()

		// 2. Gizli dosya/klasör filtresi
		if *gizliHaric && strings.HasPrefix(isim, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 3. Hariç tutulacak klasör kontrolü 
		if d.IsDir() && len(exclList) > 0 {
			isimKucuk := strings.ToLower(isim)
			for _, excl := range exclList {
				if isimKucuk == excl {
					return filepath.SkipDir
				}
			}
		}

		// 4. Derinlik Kontrolü
		if *maxDerinlik >= 0 {
			mevcutDerinlik := strings.Count(temizYol, string(os.PathSeparator)) - kokDerinlik
			if mevcutDerinlik > *maxDerinlik {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 5. Uzantı Filtresi (uzantı aratılıyorsa klasörleri atla)
		if uzantiHedef != "" {
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(isim), uzantiHedef) {
				return nil
			}
		}

		// 6. Sadece Dizin / Sadece Dosya Filtresi
		if *sadeceDizin && !d.IsDir() {
			return nil
		}
		if *sadeceDosya && d.IsDir() {
			return nil
		}

		// 7. İsim/Metin Kontrolü
		if arananKucuk != "" {
			if !strings.Contains(strings.ToLower(isim), arananKucuk) {
				return nil
			}
		}

		// Dosya/Dizin Detayı alma
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// 8. Son Değişiklik Tarihi Filtresi
		if *gunSiniri > 0 {
			sinirTarih := time.Now().AddDate(0, 0, -*gunSiniri)
			if info.ModTime().Before(sinirTarih) {
				return nil
			}
		}

		// 9. İçerik Araması (Sadece dosyalar için)
		if *icerikAra != "" {
			if d.IsDir() {
				return nil
			}
			// Maksimum 10MB boyuta kadar olan dosyaların içeriğini tara
			if !dosyaIcerikAra(yol, *icerikAra, 10*1024*1024) {
				return nil
			}
		}

		// Eşleşme Bulunması Durumu
		bulunanSayisi++

		if d.IsDir() {
			logVeYaz(fmt.Sprintf("%s[DIZIN]%s %s", ColorBlue, ColorReset, yol))
		} else {
			toplamBoyut += info.Size()
			boyutStr := formatBoyut(info.Size())
			tarihStr := info.ModTime().Format("2006-01-02 15:04")
			logVeYaz(fmt.Sprintf("%s[DOSYA]%s %-45s %s(%s)%s %s[%s]%s",
				ColorGreen, ColorReset, yol, ColorYellow, boyutStr, ColorReset, ColorGray, tarihStr, ColorReset))
		}

		return nil
	})

	if err != nil {
		logVeYaz(fmt.Sprintf("%sTarama sırasında hata oluştu: %v%s", ColorRed, err, ColorReset))
	}

	gecenSure := time.Since(baslangic)
	logVeYaz(strings.Repeat("-", 60))
	logVeYaz(fmt.Sprintf("%s✅ İşlem Tamamlandı!%s", ColorGreen, ColorReset))
	logVeYaz(fmt.Sprintf("📊 Taranan Toplam Öğe : %d", toplamTarama))
	logVeYaz(fmt.Sprintf("🎯 Bulunan Eşleşmeler : %d", bulunanSayisi))
	logVeYaz(fmt.Sprintf("💾 Toplam Eşleşen Boyut: %s", formatBoyut(toplamBoyut)))
	logVeYaz(fmt.Sprintf("⏱️ Geçen Süre         : %v", gecenSure))

	if *ciktiDosyasi != "" {
		fmt.Printf("\n📄 Sonuçlar %s'%s'%s dosyasına kaydedildi.\n", ColorYellow, *ciktiDosyasi, ColorReset)
	}
}
