//go:build ignore
package main
//Это тот же файл flags, только целиком, для запуска локально

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"

    "gitfame"



)

type Config struct {
	Repository   string
	Revision     string
	OrderBy      string
	UseCommitter bool
	Format       string
	Extensions   []string
	Languages    []string
	Exclude      []string
	RestrictTo   []string
}


type LanguageEntry struct {
    Name       string   `json:"name"`
    Type       string   `json:"type"`
    Extensions []string `json:"extensions"`
}

func loadLanguageMap() (map[string]string, error) {

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

func warnUnknownLanguages(langs []string, langMap map[string]string) {
    // langMap: ".go" → "Go"
    // Нужно построить множество известных языков
    known := make(map[string]bool)
    for _, lang := range langMap {
        known[strings.ToLower(lang)] = true
    }

    for _, lang := range langs {
        if !known[strings.ToLower(lang)] {
            fmt.Fprintf(os.Stderr, "Warning! unknown language: %s\n", lang)
        }
    }
}



func Parse() (*Config, error) {
	cfg := &Config{}

	// --- объявление флагов ---
	pflag.StringVar(&cfg.Repository, "repository", ".", "Path to git repository")
	pflag.StringVar(&cfg.Revision, "revision", "HEAD", "Git revision")
	pflag.StringVar(&cfg.OrderBy, "order-by", "lines", "Sort key: lines, commits, files")
	pflag.BoolVar(&cfg.UseCommitter, "use-committer", false, "Use committer instead of author")
	pflag.StringVar(&cfg.Format, "format", "tabular", "Output format: tabular, csv, json, json-lines")

	var extensions []string
	var languages []string
	var exclude []string
	var restrict []string

	pflag.StringSliceVar(&extensions, "extensions", nil, "File extensions filter")
	pflag.StringSliceVar(&languages, "languages", nil, "Languages filter")
	pflag.StringSliceVar(&exclude, "exclude", nil, "Exclude glob patterns")
	pflag.StringSliceVar(&restrict, "restrict-to", nil, "Restrict-to glob patterns")

	pflag.Parse()

	cfg.Extensions = extensions
	cfg.Languages = languages
	cfg.Exclude = exclude
	cfg.RestrictTo = restrict

	fmt.Println(os.Args)

	cfg.Extensions = normalizeExtensions(extensions)
	cfg.Languages = languages
	cfg.Exclude = exclude
	cfg.RestrictTo = restrict

	fmt.Printf("Repository: %s\n", cfg.Repository)
	fmt.Printf("Revision: %s\n", cfg.Revision)
	fmt.Printf("OrderBy: %s\n", cfg.OrderBy)
	fmt.Printf("UseCommitter: %v\n", cfg.UseCommitter)
	fmt.Printf("Format: %s\n", cfg.Format)
	fmt.Printf("Extensions: %#v\n", cfg.Extensions)
	fmt.Printf("Languages: %#v\n", cfg.Languages)
	fmt.Printf("Exclude: %#v\n", cfg.Exclude)
	fmt.Printf("RestrictTo: %#v\n", cfg.RestrictTo)

	// --- валидация ---
	if err := validateOrderBy(cfg.OrderBy); err != nil {
		return nil, err
	}
	if err := validateFormat(cfg.Format); err != nil {
		return nil, err
	}

	// --- загрузка language map ---
	langMap, err := loadLanguageMap()
	if err != nil {
		return nil, err
	}

	warnUnknownLanguages(cfg.Languages, langMap)
	return cfg, nil
}

func validateOrderBy(v string) error {
	switch v {
	case "lines", "commits", "files":
		return nil
	}
	return fmt.Errorf("invalid --order-by: %s", v)
}

func validateFormat(v string) error {
	switch v {
	case "tabular", "csv", "json", "json-lines":
		return nil
	}
	return fmt.Errorf("invalid --format: %s", v)
}

func normalizeExtensions(exts []string) []string {
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, e)
	}
	return out
}

func main() {
	cfg, err := Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed config:\n%+v\n", cfg)
}
