package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ParseResult — output Parse() — extracted plain text + sourceType label untuk DB.
type ParseResult struct {
	Text       string
	SourceType string
}

// Parse takes the raw bytes of an uploaded file + its filename, and returns
// extracted plain text. Dispatches by file extension; defaults to UTF-8 plain
// text for unknown types so user nggak stuck di edge case.
//
// Supported (Minggu 4): .txt, .md, .markdown, .pdf, .docx, .pasted (no ext = treated as text).
func Parse(filename string, data []byte) (ParseResult, error) {
	if len(data) == 0 {
		return ParseResult{}, fmt.Errorf("empty file")
	}

	ext := strings.ToLower(extOf(filename))
	switch ext {
	case ".txt", ".md", ".markdown", "":
		return ParseResult{Text: cleanText(string(data)), SourceType: sourceTypeFor(ext)}, nil

	case ".pdf":
		text, err := parsePDF(data)
		if err != nil {
			return ParseResult{}, fmt.Errorf("parse pdf: %w", err)
		}
		return ParseResult{Text: cleanText(text), SourceType: "pdf"}, nil

	case ".docx":
		text, err := parseDOCX(data)
		if err != nil {
			return ParseResult{}, fmt.Errorf("parse docx: %w", err)
		}
		return ParseResult{Text: cleanText(text), SourceType: "docx"}, nil

	default:
		return ParseResult{}, fmt.Errorf("unsupported file type %q (allowed: .txt, .md, .pdf, .docx)", ext)
	}
}

// ParsePastedText — bypass file parsing untuk text yang user paste langsung.
func ParsePastedText(text string) ParseResult {
	return ParseResult{Text: cleanText(text), SourceType: "paste"}
}

// --- PDF ---

func parsePDF(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}

	var buf strings.Builder
	totalPages := r.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			// Skip bad page tapi lanjut ekstrak yang lain — beberapa PDF punya
			// page yang gagal di-parse (font issues).
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}

// --- DOCX ---
//
// DOCX = zip dengan word/document.xml di dalamnya. Text content di <w:t> tags.
// Hand-rolled minimal parser supaya nggak nambah dep besar.

type docxBody struct {
	Paragraphs []docxParagraph `xml:"body>p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func parseDOCX(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}

	var documentXML *zip.File
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			documentXML = f
			break
		}
	}
	if documentXML == nil {
		return "", fmt.Errorf("word/document.xml not found (corrupted docx?)")
	}

	rc, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("open document.xml: %w", err)
	}
	defer rc.Close()

	raw, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("read document.xml: %w", err)
	}

	// XML namespace di docx (w:body, w:p, w:t) — pakai decoder yang abaikan ns
	// via DefaultSpace trick. Cleaner: stream decode dan kumpulkan <w:t> CharData.
	var buf strings.Builder
	dec := xml.NewDecoder(bytes.NewReader(raw))

	var inParagraph bool
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("xml decode: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "t":
				// Read CharData inside <w:t>
				var text string
				if err := dec.DecodeElement(&text, &t); err == nil {
					buf.WriteString(text)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "p" && inParagraph {
				buf.WriteString("\n")
				inParagraph = false
			}
		}
	}

	return buf.String(), nil
}

// --- Helpers ---

func extOf(filename string) string {
	i := strings.LastIndex(filename, ".")
	if i < 0 {
		return ""
	}
	return filename[i:]
}

func sourceTypeFor(ext string) string {
	switch ext {
	case ".md", ".markdown":
		return "md"
	case ".txt":
		return "txt"
	}
	return "txt"
}

// cleanText collapses excessive whitespace (>2 newlines → 2, tabs → space)
// supaya chunking lebih predictable. Tidak strip semua whitespace.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", "  ")

	// Collapse 3+ newlines ke 2.
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(s)
}
