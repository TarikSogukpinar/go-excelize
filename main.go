package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

type SearchResult struct {
	Filename string `json:"filename"`
	Sheet    string `json:"sheet"`
	Cell     string `json:"cell"`
	Content  string `json:"content"`
}

type FileIndex struct {
	Filename    string    `json:"filename"`
	LastUpdated time.Time `json:"last_updated"`
}

var fileIndexMap = make(map[string]FileIndex)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("XLSX Search simple app!")
	})

	// Tüm dosyaları indeksleme endpoint'i
	app.Get("/index", func(c *fiber.Ctx) error {
		indexXLSXFiles("./xlsx_files")
		return c.JSON(fiber.Map{
			"message": "All files indexed",
			"files":   fileIndexMap,
		})
	})

	// Yeni dosyaları kontrol eden endpoint
	app.Get("/check-new", func(c *fiber.Ctx) error {
		newFiles := checkForNewFiles("./xlsx_files")
		return c.JSON(fiber.Map{
			"message":  "New files checked",
			"newFiles": newFiles,
		})
	})

	app.Get("/search", searchXLSX)

	log.Fatal(app.Listen(":3000"))
}

func indexXLSXFiles(directory string) {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("File access error %s: %v", path, err)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xlsx") {
			fileIndexMap[info.Name()] = FileIndex{
				Filename:    info.Name(),
				LastUpdated: info.ModTime(),
			}
			log.Printf("File indexed: %s", info.Name())
		}
		return nil
	})
	if err != nil {
		log.Printf("Error for searching files: %v", err)
	}
}

func checkForNewFiles(directory string) []FileIndex {
	var newFiles []FileIndex

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("File access error %s: %v", path, err)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xlsx") {

			if indexedFile, exists := fileIndexMap[info.Name()]; exists {

				if info.ModTime().After(indexedFile.LastUpdated) {
					log.Printf("File updated: %s", info.Name())
					fileIndexMap[info.Name()] = FileIndex{
						Filename:    info.Name(),
						LastUpdated: info.ModTime(),
					}
					newFiles = append(newFiles, fileIndexMap[info.Name()])
				}
			} else {
				// Yeni dosya bulunursa, indekse ekle ve listeye ekle
				log.Printf("New file found: %s", info.Name())
				fileIndexMap[info.Name()] = FileIndex{
					Filename:    info.Name(),
					LastUpdated: info.ModTime(),
				}
				newFiles = append(newFiles, fileIndexMap[info.Name()])
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error file searching: %v", err)
	}

	return newFiles
}

func searchXLSX(c *fiber.Ctx) error {
	searchText := c.Query("text")
	if searchText == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Text query parameter is required",
		})
	}

	directory := "./xlsx_files"
	var results []SearchResult

	log.Printf("Search starting. Searching text: %s", searchText)

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error file searching %s: %v", path, err)
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".xlsx") {
			log.Printf("File searching: %s", path)
			fileResults, err := searchInFile(path, searchText)
			if err != nil {
				log.Printf("Error file searching %s: %v", path, err)
				return nil
			}
			results = append(results, fileResults...)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error file searching: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error file searching: %v", err),
		})
	}

	log.Printf("Search is done. Bulunan sonuç sayısı: %d", len(results))

	if len(results) == 0 {
		return c.JSON([]SearchResult{})
	}

	return c.JSON(results)
}

func searchInFile(filePath, searchText string) ([]SearchResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("file cant open: %w", err)
	}
	defer f.Close()

	var results []SearchResult

	for _, sheetName := range f.GetSheetList() {
		log.Printf("Page searching: %s in %s", sheetName, filePath)
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Page do not read %s in %s: %v", sheetName, filePath, err)
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
					log.Printf("Access to file: %s in %s, cell %s", sheetName, filePath, cellName)
				}
			}
		}
	}

	return results, nil
}
