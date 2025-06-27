package providers

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"doc-to-text/pkg/interfaces"
	"doc-to-text/pkg/types"
)

// HTMLExtractor handles HTML and MHTML files
type HTMLExtractor struct {
	name string
}

// NewHTMLExtractor creates a new HTML extractor
func NewHTMLExtractor() interfaces.Extractor {
	return &HTMLExtractor{
		name: "html",
	}
}

// Extract extracts text from HTML/MHTML files
func (e *HTMLExtractor) Extract(ctx context.Context, inputFile string) (string, error) {
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	content, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	htmlContent := string(content)

	// Check if it's MHTML and extract HTML content
	if strings.Contains(strings.ToLower(inputFile), "mhtml") || strings.Contains(strings.ToLower(inputFile), "mht") {
		htmlContent = e.extractHTMLFromMHTML(htmlContent)
	}

	// Parse and extract text from HTML
	text, err := e.extractTextFromHTML(htmlContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	return text, nil
}

// SupportsFile checks if this extractor supports the given file type
func (e *HTMLExtractor) SupportsFile(fileInfo *types.FileInfo) bool {
	ext := strings.ToLower(fileInfo.Extension)
	return ext == "html" || ext == "htm" || ext == "mhtml" || ext == "mht"
}

// Name returns the name of the extractor
func (e *HTMLExtractor) Name() string {
	return e.name
}

// extractTextFromHTML extracts readable text from HTML content
func (e *HTMLExtractor) extractTextFromHTML(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	var textBuilder strings.Builder
	e.extractTextFromNode(doc, &textBuilder)

	text := textBuilder.String()
	text = e.cleanupText(text)

	return text, nil
}

// extractTextFromNode recursively extracts text from HTML nodes
func (e *HTMLExtractor) extractTextFromNode(node *html.Node, textBuilder *strings.Builder) {
	// Skip script and style elements completely
	if node.Type == html.ElementNode {
		if node.DataAtom == atom.Script || node.DataAtom == atom.Style {
			return
		}

		// Add spacing before block elements
		if e.isBlockElement(node.DataAtom) {
			textBuilder.WriteString("\n")
		}

		// Add space before inline elements if needed
		if node.DataAtom == atom.A || node.DataAtom == atom.Span {
			current := textBuilder.String()
			if len(current) > 0 {
				lastChar := current[len(current)-1]
				if lastChar != ' ' && lastChar != '\n' {
					textBuilder.WriteString(" ")
				}
			}
		}
	}

	// If it's a text node, add the text
	if node.Type == html.TextNode {
		text := strings.TrimSpace(node.Data)
		if text != "" {
			textBuilder.WriteString(text)
		}
	}

	// Recursively process child nodes
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		e.extractTextFromNode(child, textBuilder)
	}

	// Add spacing after block elements
	if node.Type == html.ElementNode && e.isBlockElement(node.DataAtom) {
		textBuilder.WriteString("\n")
	}
}

// isBlockElement checks if an HTML element is a block-level element
func (e *HTMLExtractor) isBlockElement(atomType atom.Atom) bool {
	blockElements := map[atom.Atom]bool{
		atom.P:          true,
		atom.Div:        true,
		atom.H1:         true,
		atom.H2:         true,
		atom.H3:         true,
		atom.H4:         true,
		atom.H5:         true,
		atom.H6:         true,
		atom.Blockquote: true,
		atom.Pre:        true,
		atom.Article:    true,
		atom.Section:    true,
		atom.Header:     true,
		atom.Footer:     true,
		atom.Nav:        true,
		atom.Aside:      true,
		atom.Main:       true,
		atom.Ul:         true,
		atom.Ol:         true,
		atom.Li:         true,
		atom.Table:      true,
		atom.Tr:         true,
		atom.Td:         true,
		atom.Th:         true,
		atom.Form:       true,
		atom.Fieldset:   true,
		atom.Address:    true,
	}

	return blockElements[atomType]
}

// cleanupText cleans up extracted text
func (e *HTMLExtractor) cleanupText(text string) string {
	// Decode HTML entities
	text = html.UnescapeString(text)

	// Replace multiple whitespace characters with single spaces
	re := regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	// Clean up excessive newlines while preserving paragraph structure
	re = regexp.MustCompile(`\n\s*\n\s*\n+`)
	text = re.ReplaceAllString(text, "\n\n")

	// Remove spaces around newlines
	re = regexp.MustCompile(` *\n *`)
	text = re.ReplaceAllString(text, "\n")

	// Ensure proper spacing after periods
	re = regexp.MustCompile(`\.([A-Z])`)
	text = re.ReplaceAllString(text, ". $1")

	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// extractHTMLFromMHTML extracts HTML content from MHTML format
func (e *HTMLExtractor) extractHTMLFromMHTML(mhtmlContent string) string {
	// Look for the main HTML content section
	htmlPattern := regexp.MustCompile(`(?i)content-type:\s*text/html[^\r\n]*\r?\n\r?\n(.*?)(?:\r?\n--|\r?\n\r?\n--|$)`)
	matches := htmlPattern.FindStringSubmatch(mhtmlContent)

	if len(matches) > 1 {
		return matches[1]
	}

	// If no specific HTML section found, return the whole content
	return mhtmlContent
}
