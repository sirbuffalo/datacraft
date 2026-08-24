package docsgen

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type page struct {
	Title, Slug string
	Order       int
	Website     bool
	Body        string
}

func Generate(sourceDirectory, outputDirectory string) error {
	entries, err := os.ReadDir(sourceDirectory)
	if err != nil {
		return err
	}
	var pages []page
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(sourceDirectory, entry.Name()))
		if err != nil {
			return err
		}
		parsed, err := parsePage(entry.Name(), string(contents))
		if err != nil {
			return err
		}
		if parsed.Website {
			pages = append(pages, parsed)
		}
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Order < pages[j].Order })
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return err
	}
	for _, current := range pages {
		rendered := renderPage(current, pages)
		if err := os.WriteFile(filepath.Join(outputDirectory, current.Slug+".html"), []byte(rendered), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func parsePage(filename, source string) (page, error) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return page{}, fmt.Errorf("%s: missing front matter", filename)
	}
	result := page{Website: true}
	index := 1
	for ; index < len(lines) && lines[index] != "---"; index++ {
		key, value, found := strings.Cut(lines[index], ":")
		if !found {
			return page{}, fmt.Errorf("%s:%d: invalid front matter", filename, index+1)
		}
		switch strings.TrimSpace(key) {
		case "title":
			result.Title = strings.TrimSpace(value)
		case "slug":
			result.Slug = strings.TrimSpace(value)
		case "order":
			result.Order, _ = strconv.Atoi(strings.TrimSpace(value))
		case "website":
			result.Website = strings.TrimSpace(value) != "false"
		}
	}
	if index == len(lines) || result.Title == "" || result.Slug == "" {
		return page{}, fmt.Errorf("%s: incomplete front matter", filename)
	}
	result.Body = strings.Join(lines[index+1:], "\n")
	return result, nil
}

func renderPage(current page, pages []page) string {
	var nav strings.Builder
	for _, item := range pages {
		currentAttribute := ""
		if item.Slug == current.Slug {
			currentAttribute = ` aria-current="page"`
		}
		fmt.Fprintf(&nav, `<a%s href="%s.html">%s</a>`, currentAttribute, item.Slug, html.EscapeString(item.Title))
	}
	content := renderMarkdown(current.Body)
	return fmt.Sprintf(`<!doctype html>
<!-- Generated from docs/*.md by go run ./cmd/docsgen. Do not edit directly. -->
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s — DataCraft</title><link rel="stylesheet" href="../styles.css"></head>
<body><main class="app-shell docs-shell"><header class="topbar"><div><p class="eyebrow">DATACRAFT V2</p><h1>%s</h1></div><a class="page-link" href="../index.html">Open compiler</a></header><nav class="docs-nav" aria-label="Documentation pages">%s</nav><section class="docs">%s</section><footer><span>Generated from Markdown</span><span>Go + WebAssembly</span></footer></main></body></html>
`, html.EscapeString(current.Title), html.EscapeString(current.Title), nav.String(), content)
}

func renderMarkdown(source string) string {
	lines := strings.Split(source, "\n")
	var output, paragraph strings.Builder
	inCode, inList, articleOpen := false, false, false
	flushParagraph := func() {
		if paragraph.Len() > 0 {
			fmt.Fprintf(&output, "<p>%s</p>\n", inline(strings.TrimSpace(paragraph.String())))
			paragraph.Reset()
		}
	}
	closeList := func() {
		if inList {
			output.WriteString("</ul>\n")
			inList = false
		}
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			flushParagraph()
			closeList()
			if inCode {
				output.WriteString("</code></pre>\n")
			} else {
				output.WriteString("<pre><code>")
			}
			inCode = !inCode
			continue
		}
		if inCode {
			output.WriteString(html.EscapeString(line) + "\n")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			flushParagraph()
			output.WriteString(`<div class="docs-intro"><h2>` + html.EscapeString(strings.TrimPrefix(line, "# ")) + "</h2></div>\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			flushParagraph()
			closeList()
			if articleOpen {
				output.WriteString("</article>\n")
			}
			if !strings.Contains(output.String(), `<div class="docs-grid">`) {
				output.WriteString(`<div class="docs-grid">` + "\n")
			}
			output.WriteString("<article><h3>" + html.EscapeString(strings.TrimPrefix(line, "## ")) + "</h3>\n")
			articleOpen = true
			continue
		}
		if strings.HasPrefix(line, "- ") {
			flushParagraph()
			if !inList {
				output.WriteString("<ul>\n")
				inList = true
			}
			output.WriteString("<li>" + inline(strings.TrimPrefix(line, "- ")) + "</li>\n")
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushParagraph()
			closeList()
			continue
		}
		if paragraph.Len() > 0 {
			paragraph.WriteByte(' ')
		}
		paragraph.WriteString(line)
	}
	flushParagraph()
	closeList()
	if articleOpen {
		output.WriteString("</article>\n</div>\n")
	}
	return output.String()
}

func inline(value string) string {
	escaped := html.EscapeString(value)
	var result strings.Builder
	for {
		before, rest, found := strings.Cut(escaped, "`")
		result.WriteString(before)
		if !found {
			break
		}
		code, after, found := strings.Cut(rest, "`")
		if !found {
			result.WriteString("`" + rest)
			break
		}
		result.WriteString("<code>" + code + "</code>")
		escaped = after
	}
	return result.String()
}
