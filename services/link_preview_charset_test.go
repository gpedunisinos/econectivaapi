package services

import (
	"golang.org/x/net/html"
	"strings"
	"testing"
)

func TestLinkPreviewDecodesDocumentCharset(t *testing.T) {
	for _, test := range []struct{ body, header string }{
		{"<title>Cidadania e educação</title>", "text/html; charset=utf-8"},
		{"<title>Cidadania e educa\xe7\xe3o</title>", "text/html; charset=iso-8859-1"},
		{"<meta charset=windows-1252><title>Cidadania e educa\xe7\xe3o</title>", "text/html"},
	} {
		node, err := parsePreviewDocument(strings.NewReader(test.body), test.header)
		if err != nil {
			t.Fatal(err)
		}
		title := ""
		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "title" {
				title = n.FirstChild.Data
			}
			for x := n.FirstChild; x != nil; x = x.NextSibling {
				walk(x)
			}
		}
		walk(node)
		if title != "Cidadania e educação" {
			t.Fatalf("incorrect encoding: %q", title)
		}
	}
}
