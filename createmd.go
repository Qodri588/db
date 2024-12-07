package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Fungsi spintax untuk menghasilkan teks unik
func parseSpintax(input string) string {
	re := regexp.MustCompile(`\{([^{}]*)\}`)
	for re.MatchString(input) {
		input = re.ReplaceAllStringFunc(input, func(match string) string {
			options := strings.Split(match[1:len(match)-1], "|")
			return options[rand.Intn(len(options))]
		})
	}
	return input
}

// Membersihkan dan menyesuaikan judul agar kompatibel dengan Hugo
func cleanTitle(title string) string {
	re := regexp.MustCompile(`[!@#$%^&*()_\-+=]`)
	title = re.ReplaceAllString(title, "")

	re = regexp.MustCompile(`@.*`)
	title = re.ReplaceAllString(title, "")

	reCek := regexp.MustCompile(`(?i)\bcek( telegram)?\b.*|link.*$`)
	title = reCek.ReplaceAllString(title, "")

	return strings.TrimSpace(title)
}

// Menyesuaikan title dengan tambahan nama folder jika terlalu pendek
func adjustShortTitle(title, folderName string) string {
	title = strings.TrimSpace(title)
	if len(title) <= 3 {
		title = fmt.Sprintf("%s %s", folderName, title)
	}
	return title
}

// Menghasilkan tanggal acak dalam format ISO 8601
func generateRandomDate() string {
	now := time.Now()
	pastDate := now.AddDate(0, 0, -180)
	randomTime := time.Unix(pastDate.Unix()+rand.Int63n(now.Unix()-pastDate.Unix()), 0)
	return randomTime.Format("2006-01-02T15:04:05-08:00")
}

// Memproses database dan menghasilkan file Markdown
func processDatabase(dbPath string) {
	// Membaca isi spintax.txt
	spintaxContent, err := os.ReadFile("spintax.txt")
	if err != nil {
		log.Fatalf("Gagal membaca file spintax.txt: %v", err)
	}

	// Membaca isi tags.txt
	tagFileContent, err := os.ReadFile("tags.txt")
	if err != nil {
		log.Fatalf("Gagal membaca file tags.txt: %v", err)
	}
	tagLines := strings.Fields(string(tagFileContent)) // Mengambil setiap baris sebagai tag

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Gagal membuka database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT file_code, title, download_url, single_img, length, views, uploaded, fld_id, name FROM files")
	if err != nil {
		log.Fatalf("Gagal mengambil data: %v", err)
	}
	defer rows.Close()

	baseFolder := "markdown"
	if _, err := os.Stat(baseFolder); os.IsNotExist(err) {
		if err := os.Mkdir(baseFolder, 0755); err != nil {
			log.Fatalf("Gagal membuat folder 'markdown': %v", err)
		}
		fmt.Println("Folder 'markdown' berhasil dibuat.")
	}

	for rows.Next() {
		var fileCode, title, downloadUrl, singleImg, fldID, name, uploaded string
		var length, views int

		if err := rows.Scan(&fileCode, &title, &downloadUrl, &singleImg, &length, &views, &uploaded, &fldID, &name); err != nil {
			log.Printf("Gagal membaca baris: %v", err)
			continue
		}

		// Bersihkan dan validasi nama folder
		cleanName := cleanTitle(name)
		folderPath := filepath.Join(baseFolder, cleanName)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			if err := os.Mkdir(folderPath, 0755); err != nil {
				log.Printf("Gagal membuat folder '%s': %v", cleanName, err)
				continue
			}
			fmt.Printf("Folder '%s' berhasil dibuat di dalam folder 'markdown'.\n", cleanName)
		}

		// Bersihkan dan validasi title
		cleanTitleValue := cleanTitle(title)
		cleanTitleValue = adjustShortTitle(cleanTitleValue, cleanName)

		// Memfilter tags dari title
		rawTags := strings.Fields(cleanTitleValue)
		var tags []string
		for _, tag := range rawTags {
			if len(tag) >= 3 && regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(tag) {
				tags = append(tags, tag)
			}
		}

		// Jika tidak ada tags, tambahkan "indo" dan tag dari tags.txt
		if len(tags) == 0 {
			tags = append(tags, "indo")
		}
		tags = append(tags, tagLines...) // Tambahkan tags dari tags.txt

		// Proses spintax dan ganti {{ .Title }} dengan cleanTitleValue
		spintaxString := strings.ReplaceAll(string(spintaxContent), "{{ .Title }}", cleanTitleValue)
		description := parseSpintax(spintaxString)

		// Menyusun frontmatter dalam format Markdown
		frontmatter := fmt.Sprintf(`--- 
title: "%s"
description: "%s"
date: %s
file_code: "%s"
draft: false
cover: "%s"
tags: ["%s"]
length: %d
fld_id: "%s"
foldername: "%s"
categories: ["%s"]
views: %d
---`,
			cleanTitleValue,
			description,
			generateRandomDate(),
			fileCode,
			strings.Replace(singleImg, "https://img.doodcdn.co/snaps/", "", 1),
			strings.Join(tags, `", "`),
			length,
			fldID,
			cleanName,
			cleanName,
			views,
		)

		mdFilename := regexp.MustCompile(`[\\/:*?"<>|]`).ReplaceAllString(cleanTitleValue, "") + ".md"
		mdPath := filepath.Join(folderPath, mdFilename)

		// Menulis file markdown ke dalam folder yang sesuai
		if err := os.WriteFile(mdPath, []byte(frontmatter), 0644); err != nil {
			log.Printf("Gagal membuat file '%s': %v", mdFilename, err)
			continue
		}
		fmt.Printf("File '%s' berhasil dibuat di folder '%s'.\n", mdFilename, folderPath)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Kesalahan iterasi baris: %v", err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	processDatabase("dood.db")
}
