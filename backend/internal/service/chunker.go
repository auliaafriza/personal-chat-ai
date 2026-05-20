package service

import (
	"strings"
)

// Chunk — single splitted slice of a Document, ready to embed.
type Chunk struct {
	Position int    // 0-based, urutan dalam document
	Heading  string // closest preceding markdown heading (empty kalau plain text)
	Content  string
}

// Default chunking parameters. Tweak via ChunkOptions kalau perlu.
const (
	DefaultMaxChars     = 1500
	DefaultMinChars     = 200
	DefaultOverlapChars = 100
)

type ChunkOptions struct {
	MaxChars     int
	MinChars     int
	OverlapChars int
}

func (o ChunkOptions) withDefaults() ChunkOptions {
	if o.MaxChars <= 0 {
		o.MaxChars = DefaultMaxChars
	}
	if o.MinChars <= 0 {
		o.MinChars = DefaultMinChars
	}
	if o.OverlapChars < 0 {
		o.OverlapChars = DefaultOverlapChars
	}
	if o.OverlapChars >= o.MaxChars {
		o.OverlapChars = o.MaxChars / 4
	}
	return o
}

// SplitChunks splits text into chunks using a heading-aware strategy:
//   1. Split text into sections by markdown headings (# / ## / ###).
//   2. Untuk setiap section, kalau body lebih dari MaxChars → fallback split
//      by char dengan overlap.
//   3. Section yang lebih kecil dari MinChars di-merge ke section sebelumnya
//      (kecuali kalau heading-nya berbeda, biar konteks tetap nyambung).
//
// Cocok untuk markdown + plain text (no headings → semua jadi 1 section yang
// otomatis fallback ke fixed-size split).
func SplitChunks(text string, opts ChunkOptions) []Chunk {
	opts = opts.withDefaults()
	if strings.TrimSpace(text) == "" {
		return nil
	}

	sections := splitByHeadings(text)
	var raw []Chunk

	for _, sec := range sections {
		if len(sec.Content) <= opts.MaxChars {
			raw = append(raw, Chunk{Heading: sec.Heading, Content: sec.Content})
			continue
		}
		// Fallback: split by chars with overlap.
		pieces := splitByChars(sec.Content, opts.MaxChars, opts.OverlapChars)
		for _, p := range pieces {
			raw = append(raw, Chunk{Heading: sec.Heading, Content: p})
		}
	}

	// Merge tiny chunks ke yang sebelumnya kalau heading sama.
	merged := mergeSmall(raw, opts.MinChars)

	// Assign positions.
	for i := range merged {
		merged[i].Position = i
	}
	return merged
}

// --- Heading split ---

type headingSection struct {
	Heading string
	Content string
}

func splitByHeadings(text string) []headingSection {
	lines := strings.Split(text, "\n")
	var sections []headingSection
	var currentHeading string
	var buf strings.Builder

	flush := func() {
		content := strings.TrimSpace(buf.String())
		if content != "" || currentHeading != "" {
			sections = append(sections, headingSection{
				Heading: currentHeading,
				Content: content,
			})
		}
		buf.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownHeading(trimmed) {
			// Tutup section sebelumnya.
			flush()
			currentHeading = strings.TrimLeft(trimmed, "# ")
			// Heading-nya tetap dimasukkan ke content (biar embedded juga).
			buf.WriteString(trimmed)
			buf.WriteString("\n")
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()

	// Edge case: empty document
	if len(sections) == 0 {
		return []headingSection{{Heading: "", Content: text}}
	}
	return sections
}

func isMarkdownHeading(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	// "# foo", "## foo", "### foo" — sampai 6 level.
	rest := strings.TrimLeft(line, "#")
	if rest == line {
		return false
	}
	return strings.HasPrefix(rest, " ") && len(line)-len(rest) <= 6
}

// --- Fixed-size split with overlap ---

func splitByChars(text string, max, overlap int) []string {
	if len(text) <= max {
		return []string{text}
	}

	var out []string
	step := max - overlap
	if step <= 0 {
		step = max
	}

	for start := 0; start < len(text); start += step {
		end := start + max
		if end >= len(text) {
			out = append(out, text[start:])
			break
		}
		// Coba potong di whitespace terdekat untuk menjaga readability.
		cut := end
		for cut > start+max/2 && !isBreakable(text[cut-1]) {
			cut--
		}
		if cut == start+max/2 {
			cut = end
		}
		out = append(out, text[start:cut])
	}
	return out
}

func isBreakable(b byte) bool {
	return b == ' ' || b == '\n' || b == '\t' || b == '.' || b == ',' || b == ';'
}

// --- Merge small chunks ---

func mergeSmall(chunks []Chunk, minChars int) []Chunk {
	if len(chunks) <= 1 {
		return chunks
	}
	out := make([]Chunk, 0, len(chunks))
	for _, c := range chunks {
		if len(out) > 0 && len(c.Content) < minChars && out[len(out)-1].Heading == c.Heading {
			// Append ke chunk sebelumnya.
			out[len(out)-1].Content += "\n\n" + c.Content
			continue
		}
		out = append(out, c)
	}
	return out
}
