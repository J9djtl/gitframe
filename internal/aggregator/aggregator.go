package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"gitfame/internal/flags"
)

// AuthorStat хранит агрегированную статистику одного автора.
type AuthorStat struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Commits int    `json:"commits"`
	Files   int    `json:"files"`
}

// commitInfo — данные об одном коммите, извлечённые из git blame --porcelain.
type commitInfo struct {
	author    string
	committer string
	lineCount int
}

// ExecuteGitFrame принимает конфиг и уже отфильтрованный список файлов,
// параллельно запускает git blame для каждого файла и возвращает
// агрегированную статистику по авторам (без сортировки).
func ExecuteGitFrame(cfg *flags.Config, files []string) ([]AuthorStat, error) {
	blameResults, err := blameAll(cfg, files)
	if err != nil {
		return nil, err
	}
	return aggregate(cfg, blameResults), nil
}

// Параллельный git blame

// fileBlame — результат git blame для одного файла:
// map[commitHash]commitInfo.
type fileBlame struct {
	filePath string
	commits  map[string]commitInfo
}

func blameAll(cfg *flags.Config, files []string) ([]fileBlame, error) {
	type result struct {
		fb  fileBlame
		err error
	}

	ch := make(chan result, len(files))
	// Семафор — не более 16 параллельных git-процессов.
	sem := make(chan struct{}, 16)

	var wg sync.WaitGroup
	for _, f := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fb, err := blameFile(cfg, path)
			ch <- result{fb, err}
		}(f)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var results []fileBlame
	for r := range ch {
		if r.err != nil {
			return nil, r.err
		}
		results = append(results, r.fb)
	}
	return results, nil
}

// blameFile запускает `git blame --porcelain <revision> -- <path>`
// и разбирает вывод в map[hash]commitInfo.
func blameFile(cfg *flags.Config, path string) (fileBlame, error) {
	fb := fileBlame{
		filePath: path,
		commits:  make(map[string]commitInfo),
	}

	out, err := runGit(cfg.Repository, "blame", "--porcelain", cfg.Revision, "--", path)
	if err != nil {
		return fb, fmt.Errorf("git blame %s: %w", path, err)
	}

	// Porcelain-формат:
	// <40-char hash> <orig_line> <final_line> [<line_count>]   ← заголовок блока
	// author <name>
	// committer <name>
	// ... (остальные поля коммита, только при первом появлении хэша)
	// \t<содержимое строки>
	// При повторном появлении уже известного хэша его поля (author, committer и т.д.)
	// не выводятся повторно — только заголовок блока со счётчиком строк.

	lines := strings.Split(out, "\n")
	currentHash := ""

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Заголовок блока
		if len(line) >= 40 && !strings.HasPrefix(line, "\t") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && len(fields[0]) == 40 {
				currentHash = fields[0]
				// 4-й токен — количество строк в блоке (есть только в заголовке блока).
				if len(fields) == 4 {
					n := parseInt(fields[3])
					ci := fb.commits[currentHash]
					ci.lineCount += n
					fb.commits[currentHash] = ci
				}
			}
			continue
		}

		if currentHash == "" {
			continue
		}

		// Поля коммита — записываем только при первом появлении хэша.
		ci := fb.commits[currentHash]

		switch {
		case strings.HasPrefix(line, "author "):
			if ci.author == "" {
				ci.author = strings.TrimPrefix(line, "author ")
			}
		case strings.HasPrefix(line, "committer "):
			if ci.committer == "" {
				ci.committer = strings.TrimPrefix(line, "committer ")
			}
		}

		fb.commits[currentHash] = ci
	}

	return fb, nil
}

// Агрегация по авторам
func aggregate(cfg *flags.Config, fileBlames []fileBlame) []AuthorStat {
	type authorData struct {
		lines   int
		commits map[string]bool
		files   map[string]bool
	}

	authors := make(map[string]*authorData)

	ensure := func(name string) {
		if _, ok := authors[name]; !ok {
			authors[name] = &authorData{
				commits: make(map[string]bool),
				files:   make(map[string]bool),
			}
		}
	}

	for _, fb := range fileBlames {
		for hash, ci := range fb.commits {
			name := ci.author
			if cfg.UseCommitter {
				name = ci.committer
			}
			if name == "" {
				name = "Unknown"
			}

			ensure(name)
			d := authors[name]
			d.lines += ci.lineCount
			d.commits[hash] = true
			d.files[fb.filePath] = true
		}
	}

	stats := make([]AuthorStat, 0, len(authors))
	for name, d := range authors {
		stats = append(stats, AuthorStat{
			Name:    name,
			Lines:   d.lines,
			Commits: len(d.commits),
			Files:   len(d.files),
		})
	}
	return stats
}

// Вспомогательные функции
func runGit(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}

