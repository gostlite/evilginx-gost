package core

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// InjectJavaScriptHTML injects JavaScript into HTML using proper HTML parsing
// location can be: "head", "body_top", "body_bottom"
func InjectJavaScriptHTML(htmlContent []byte, script string, location string) ([]byte, error) {
	// Parse HTML
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		// Fallback to original content if parsing fails
		return htmlContent, fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Create script node
	scriptNode := createScriptNode(script)

	// Find injection point and inject
	injected := false
	switch location {
	case "head":
		injected = injectIntoHead(doc, scriptNode)
	case "body_top":
		injected = injectIntoBodyTop(doc, scriptNode)
	case "body_bottom":
		injected = injectIntoBodyBottom(doc, scriptNode)
	default:
		return htmlContent, fmt.Errorf("invalid injection location: %s", location)
	}

	if !injected {
		// If injection failed, return original content
		return htmlContent, fmt.Errorf("failed to find injection point for location: %s", location)
	}

	// Render HTML back to bytes
	var buf bytes.Buffer
	err = html.Render(&buf, doc)
	if err != nil {
		return htmlContent, fmt.Errorf("failed to render HTML: %v", err)
	}

	return buf.Bytes(), nil
}

// createScriptNode creates a script element node
func createScriptNode(script string) *html.Node {
	scriptElem := &html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{
			{Key: "type", Val: "text/javascript"},
		},
	}
	
	// Add script content as text node
	scriptText := &html.Node{
		Type: html.TextNode,
		Data: script,
	}
	scriptElem.AppendChild(scriptText)
	
	return scriptElem
}

// injectIntoHead injects script into <head> element
func injectIntoHead(doc *html.Node, scriptNode *html.Node) bool {
	head := findNode(doc, "head")
	if head == nil {
		return false
	}
	
	// Append to end of head
	head.AppendChild(scriptNode)
	return true
}

// injectIntoBodyTop injects script at the beginning of <body>
func injectIntoBodyTop(doc *html.Node, scriptNode *html.Node) bool {
	body := findNode(doc, "body")
	if body == nil {
		return false
	}
	
	// Insert as first child of body
	if body.FirstChild != nil {
		body.InsertBefore(scriptNode, body.FirstChild)
	} else {
		body.AppendChild(scriptNode)
	}
	return true
}

// injectIntoBodyBottom injects script at the end of <body>
func injectIntoBodyBottom(doc *html.Node, scriptNode *html.Node) bool {
	body := findNode(doc, "body")
	if body == nil {
		return false
	}
	
	// Append to end of body
	body.AppendChild(scriptNode)
	return true
}

// findNode recursively searches for a node with the given tag name
func findNode(n *html.Node, tagName string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tagName {
		return n
	}
	
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findNode(c, tagName); result != nil {
			return result
		}
	}
	
	return nil
}

// HasHTMLStructure checks if content has basic HTML structure (html, head, body tags)
func HasHTMLStructure(content []byte) bool {
	contentLower := strings.ToLower(string(content))
	return strings.Contains(contentLower, "<html") && 
	       (strings.Contains(contentLower, "<head") || strings.Contains(contentLower, "<body"))
}

// ObfuscateJavaScript applies obfuscation to JavaScript based on level
// Levels: off, low, medium, high, ultra
func ObfuscateJavaScript(script string, level string) string {
	switch level {
	case "off":
		return script
	case "low":
		// Basic: minify whitespace
		return minifyWhitespace(script)
	case "medium":
		// Medium: minify + encode strings
		return encodeStrings(minifyWhitespace(script))
	case "high":
		// High: medium + variable renaming
		return renameVariables(encodeStrings(minifyWhitespace(script)))
	case "ultra":
		// Ultra: high + control flow flattening (basic)
		return flattenControlFlow(renameVariables(encodeStrings(minifyWhitespace(script))))
	default:
		return script
	}
}

// minifyWhitespace removes unnecessary whitespace
func minifyWhitespace(script string) string {
	// Simple minification: remove extra spaces, newlines, tabs
	script = strings.ReplaceAll(script, "\n", " ")
	script = strings.ReplaceAll(script, "\r", "")
	script = strings.ReplaceAll(script, "\t", " ")
	
	// Remove multiple spaces
	for strings.Contains(script, "  ") {
		script = strings.ReplaceAll(script, "  ", " ")
	}
	
	return strings.TrimSpace(script)
}

// encodeStrings encodes string literals (basic implementation)
func encodeStrings(script string) string {
	// For now, return as-is. Full implementation would encode string literals
	// This is a placeholder for more sophisticated obfuscation
	return script
}

// renameVariables renames variables to short names (basic implementation)
func renameVariables(script string) string {
	// Placeholder for variable renaming
	// Full implementation would parse AST and rename variables
	return script
}

// flattenControlFlow flattens control flow (basic implementation)
func flattenControlFlow(script string) string {
	// Placeholder for control flow flattening
	// Full implementation would restructure if/else, loops, etc.
	return script
}
