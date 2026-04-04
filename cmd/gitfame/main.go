//go:build !solution

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gitfame"
	"gitfame/internal/flags"
)

type LanguageEntry struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Extensions []string `json:"extensions"`
}

func loadLanguageMap() (map[string]string, error) {

	// Функция парсит language_extensions.json и возвращает словарь  langMap: ".go" → "Go"

	data, err := gitfame.ConfigFS.ReadFile("configs/language_extensions.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read language_extensions.json: %w", err)
	}

	var arr []LanguageEntry
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("failed to parse language_extensions.json: %w", err)
	}

	m := make(map[string]string)
	for _, entry := range arr {
		for _, ext := range entry.Extensions {
			m[ext] = entry.Name
		}
	}

	return m, nil
}

func main() {

	//парсим языки
	langMap, err := loadLanguageMap()
	if err != nil {
		log.Fatal(err)
	}

	//парсим флаги и выходим с 1, если некорректны
	cfg, err := flags.Parse(langMap)
	fmt.Println(cfg)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid flags: %s\n", err)
		os.Exit(1)
	}

	// получаем множестов файлов репозитория и используя собранные флаги фильтруем их

	// используя полученное множество файлов, проходимся по ним построчно ( параллельно )
	// и считаем статистики

	//выводим статистики в нужном формате
}
