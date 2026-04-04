
package flags

//модуль парсинга флагов

import (
    "fmt"
    "os"
    "strings"

    "github.com/spf13/pflag"
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

func warnUnknownLanguages(langs []string, langMap map[string]string) {
    
    // Нужно построить множество известных языков
    known := make(map[string]bool)
    for _, lang := range langMap {
        known[strings.ToLower(lang)] = true
    }

    for _, lang := range langs {
        if !known[strings.ToLower(lang)] {
            fmt.Fprintf(os.Stderr, "Warning: unknown language: %s\n", lang)
        }
    }
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


func Parse(langMap map[string]string) (*Config, error) {
    cfg := &Config{}

    pflag.StringVar(&cfg.Repository, "repository", ".", "Path to git repository")
    pflag.StringVar(&cfg.Revision, "revision", "HEAD", "Git revision")
    pflag.StringVar(&cfg.OrderBy, "order-by", "lines", "Sort key: lines, commits, files")
    pflag.BoolVar(&cfg.UseCommitter, "use-committer", false, "Use committer instead of author")
    pflag.StringVar(&cfg.Format, "format", "tabular", "Output format: tabular, csv, json, json-lines")

    var(
        extensions []string
        languages []string
        exclude []string
        restrict []string
    )
    

    pflag.StringSliceVar(&extensions, "extensions", nil, "File extensions filter")
    pflag.StringSliceVar(&languages, "languages", nil, "Languages filter")
    pflag.StringSliceVar(&exclude, "exclude", nil, "Exclude glob patterns")
    pflag.StringSliceVar(&restrict, "restrict-to", nil, "Restrict-to glob patterns")

    pflag.Parse()

    cfg.Extensions = normalizeExtensions(extensions)
    cfg.Languages = languages
    cfg.Exclude = exclude
    cfg.RestrictTo = restrict

    if err := validateOrderBy(cfg.OrderBy); err != nil {
        return nil, err
    }
    if err := validateFormat(cfg.Format); err != nil {
        return nil, err
    }

    warnUnknownLanguages(cfg.Languages, langMap)

    return cfg, nil
}
