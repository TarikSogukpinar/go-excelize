package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

type SearchResult struct {
	Filename string `json:"filename"`
	Sheet    string `json:"sheet"`
	Cell     string `json:"cell"`
	Content  string `json:"content"`
}

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("XLSX Metin Arama Uygulamasına Hoş Geldiniz!")
	})

	app.Get("/search", searchXLSX)

	log.Fatal(app.Listen(":3000"))
}

func searchXLSX(c *fiber.Ctx) error {
	searchText := c.Query("text")
	if searchText == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Arama metni belirtilmedi",
		})
	}

	directory := "./xlsx_files"
	var results []SearchResult

	log.Printf("Arama başlatılıyor. Aranacak metin: %s", searchText)

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Dosya erişim hatası %s: %v", path, err)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xlsx") {
			log.Printf("Dosya taranıyor: %s", path)
			fileResults, err := searchInFile(path, searchText)
			if err != nil {
				log.Printf("Dosya aranırken hata oluştu %s: %v", path, err)
				return nil
			}
			results = append(results, fileResults...)
		}
		return nil
	})

	if err != nil {
		log.Printf("Dizin taranırken hata oluştu: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Dizin taranırken hata oluştu: %v", err),
		})
	}

	log.Printf("Arama tamamlandı. Bulunan sonuç sayısı: %d", len(results))

	if len(results) == 0 {
		return c.JSON([]SearchResult{})
	}

	return c.JSON(results)
}

func searchInFile(filePath, searchText string) ([]SearchResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("dosya açılamadı: %w", err)
	}
	defer f.Close()

	var results []SearchResult

	for _, sheetName := range f.GetSheetList() {
		log.Printf("Sayfa taranıyor: %s in %s", sheetName, filePath)
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Sayfa okunamadı %s in %s: %v", sheetName, filePath, err)
			continue
		}

		for rowIndex, row := range rows {
			for colIndex, cell := range row {
				if strings.Contains(strings.ToLower(cell), strings.ToLower(searchText)) {
					cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
					if err != nil {
						cellName = fmt.Sprintf("R%dC%d", rowIndex+1, colIndex+1)
					}
					results = append(results, SearchResult{
						Filename: filepath.Base(filePath),
						Sheet:    sheetName,
						Cell:     cellName,
						Content:  cell,
					})
					log.Printf("Eşleşme bulundu: %s in %s, hücre %s", sheetName, filePath, cellName)
				}
			}
		}
	}

	return results, nil
}
