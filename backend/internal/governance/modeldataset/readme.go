// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modeldataset

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
)

const MaxStoredReadmeBytes = 64 * 1024

var (
	readmeHTMLTablePattern       = regexp.MustCompile(`(?is)<table\b[^>]*>.*?</table>`)
	readmeHTMLTagPattern         = regexp.MustCompile(`<[^>]+>`)
	readmeUnsafeHTMLBlockPattern = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
)

// CleanReadme removes source-controlled raw HTML that the Markdown renderer does
// not support, converts simple HTML tables to Markdown, and applies a UTF-8-safe
// byte limit before the content is persisted.
func CleanReadme(text string, limit int) string {
	if limit < 1 {
		return ""
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---"); end >= 0 {
			text = text[end+8:]
		}
	}
	text = readmeUnsafeHTMLBlockPattern.ReplaceAllString(text, "")
	text = readmeHTMLTablePattern.ReplaceAllStringFunc(text, htmlTableToMarkdown)
	text = readmeHTMLTagPattern.ReplaceAllString(text, " ")
	text = html.UnescapeString(text)
	return truncateReadme(strings.TrimSpace(text), limit)
}

func truncateReadme(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	for limit > 0 && !utf8.RuneStart(text[limit]) {
		limit--
	}
	return text[:limit]
}

func htmlTableToMarkdown(tableHTML string) string {
	document, err := xhtml.Parse(strings.NewReader(tableHTML))
	if err != nil {
		return tableHTML
	}
	table := findHTMLNode(document, "table")
	if table == nil {
		return tableHTML
	}

	rows := make([][]string, 0)
	collectHTMLTableRows(table, &rows)
	columnCount := 0
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if len(rows) == 0 || columnCount == 0 {
		return tableHTML
	}

	var result strings.Builder
	result.WriteString("\n\n")
	writeMarkdownTableRow(&result, rows[0], columnCount)
	separator := make([]string, columnCount)
	for i := range separator {
		separator[i] = "---"
	}
	writeMarkdownTableRow(&result, separator, columnCount)
	for _, row := range rows[1:] {
		writeMarkdownTableRow(&result, row, columnCount)
	}
	result.WriteString("\n")
	return result.String()
}

func findHTMLNode(node *xhtml.Node, tag string) *xhtml.Node {
	if node.Type == xhtml.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLNode(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func collectHTMLTableRows(node *xhtml.Node, rows *[][]string) {
	if node.Type == xhtml.ElementNode && node.Data == "tr" {
		row := make([]string, 0)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != xhtml.ElementNode || child.Data != "th" && child.Data != "td" {
				continue
			}
			cell := strings.Join(strings.Fields(htmlNodeText(child)), " ")
			cell = strings.ReplaceAll(cell, "|", `\|`)
			row = append(row, cell)
			for i := 1; i < htmlColSpan(child); i++ {
				row = append(row, "")
			}
		}
		if len(row) > 0 {
			*rows = append(*rows, row)
		}
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		collectHTMLTableRows(child, rows)
	}
}

func htmlNodeText(node *xhtml.Node) string {
	if node.Type == xhtml.TextNode {
		return node.Data
	}
	var result strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result.WriteString(htmlNodeText(child))
		result.WriteByte(' ')
	}
	return result.String()
}

func htmlColSpan(node *xhtml.Node) int {
	for _, attribute := range node.Attr {
		if attribute.Key == "colspan" {
			span, err := strconv.Atoi(attribute.Val)
			if err == nil && span > 1 {
				return span
			}
		}
	}
	return 1
}

func writeMarkdownTableRow(result *strings.Builder, row []string, columnCount int) {
	result.WriteString("| ")
	for column := 0; column < columnCount; column++ {
		if column < len(row) {
			result.WriteString(row[column])
		}
		result.WriteString(" |")
		if column < columnCount-1 {
			result.WriteByte(' ')
		}
	}
	result.WriteByte('\n')
}
