package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <dist-dir>\n", os.Args[0])
		os.Exit(1)
	}

	distDir := os.Args[1]
	indexPath := filepath.Join(distDir, "index.html")

	// Read README.md
	readmeContent, err := os.ReadFile("README.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading README.md: %v\n", err)
		os.Exit(1)
	}

	// Convert README to HTML
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(readmeContent)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)
	readmeHTML := markdown.Render(doc, renderer)

	// Detect versions from files in dist directory (flat structure from goreleaser)
	version := detectVersionFromDist(distDir)

	// Generate downloads HTML with relative paths
	downloadsHTML := generateDownloadsHTMLFlat(distDir, version)

	// Replace installation section with downloads table
	readmeHTML = replaceInstallationSection(readmeHTML, downloadsHTML)

	// Create index.html
	f, err := os.Create(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating index.html: %v\n", err)
		os.Exit(1)
	}

	// Write HTML header
	writeHeader(f)

	// Write README content (with downloads section embedded)
	if _, err := f.Write(readmeHTML); err != nil {
		f.Close()
		fmt.Fprintf(os.Stderr, "Error writing README content: %v\n", err)
		os.Exit(1)
	}

	// Write HTML footer
	writeFooter(f)
	f.Close()

	fmt.Fprintf(os.Stderr, "Generated %s\n", indexPath)
}

// detectVersionFromDist finds the version string from files like kvx_0.1.0-SNAPSHOT-abc123_Darwin_arm64.tar.gz
func detectVersionFromDist(distDir string) string {
	files, err := os.ReadDir(distDir)
	if err != nil {
		return "unknown"
	}

	// Pattern: kvx_VERSION_OS_ARCH.ext
	re := regexp.MustCompile(`^kvx_([^_]+(?:-[^_]+)*)_(?:Darwin|Linux|Windows)_(?:arm64|x86_64)\.(?:tar\.gz|zip)$`)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(file.Name())
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return "unknown"
}

// generateDownloadsHTMLFlat generates download links for flat dist/ directory structure
func generateDownloadsHTMLFlat(distDir, version string) string {
	var sb strings.Builder

	sb.WriteString(`  <div class="downloads">
    <h2>📦 Downloads</h2>
    <div class="version-section">
`)
	sb.WriteString(fmt.Sprintf(`      <h3>%s</h3>
      <table class="download-table">
`, version))

	files, err := os.ReadDir(distDir)
	if err != nil {
		sb.WriteString(`      </table>
    </div>
  </div>
`)
		return sb.String()
	}

	// Group files by platform
	type Platform struct {
		Name    string
		Archive string
	}
	platforms := make(map[string]*Platform)

	for _, file := range files {
		name := file.Name()
		// Skip non-archives
		if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".zip") {
			continue
		}
		// Skip checksums
		if strings.Contains(name, "SHA256") {
			continue
		}

		// Detect platform
		var platKey, platName string
		switch {
		case strings.Contains(name, "Darwin_arm64") || strings.Contains(name, "darwin_arm64"):
			platKey = "darwin-arm64"
			platName = "macOS (Apple Silicon)"
		case strings.Contains(name, "Darwin_x86_64") || strings.Contains(name, "darwin_amd64"):
			platKey = "darwin-amd64"
			platName = "macOS (Intel)"
		case strings.Contains(name, "Linux_arm64") || strings.Contains(name, "linux_arm64"):
			platKey = "linux-arm64"
			platName = "Linux (ARM64)"
		case strings.Contains(name, "Linux_x86_64") || strings.Contains(name, "linux_amd64"):
			platKey = "linux-amd64"
			platName = "Linux (x86_64)"
		case strings.Contains(name, "Windows_arm64") || strings.Contains(name, "windows_arm64"):
			platKey = "windows-arm64"
			platName = "Windows (ARM64)"
		case strings.Contains(name, "Windows_x86_64") || strings.Contains(name, "windows_amd64"):
			platKey = "windows-amd64"
			platName = "Windows (x86_64)"
		default:
			continue
		}

		if platforms[platKey] == nil {
			platforms[platKey] = &Platform{Name: platName, Archive: name}
		}
	}

	// Sort platforms for consistent output
	platKeys := make([]string, 0, len(platforms))
	for k := range platforms {
		platKeys = append(platKeys, k)
	}
	sort.Strings(platKeys)

	for _, key := range platKeys {
		plat := platforms[key]
		// Use relative path (just the filename since index.html is in same directory)
		sb.WriteString(fmt.Sprintf(`        <tr>
          <td class="platform-name">%s</td>
          <td class="platform-links"><a href="%s">download</a></td>
        </tr>
`, plat.Name, plat.Archive))
	}

	sb.WriteString(`      </table>
    </div>
  </div>
`)

	return sb.String()
}

func writeHeader(w io.Writer) {
	fmt.Fprint(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>kvx - Interactive Data Explorer</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; line-height: 1.6; color: #333; }
    h1 { color: #2563eb; border-bottom: 2px solid #2563eb; padding-bottom: 10px; }
    h2 { color: #1e40af; margin-top: 30px; }
    h3 { color: #1e3a8a; margin-top: 20px; }
    code { background: #f1f5f9; padding: 2px 6px; border-radius: 3px; font-family: Monaco, Menlo, monospace; font-size: 0.9em; }
    pre { background: #1e293b; color: #e2e8f0; padding: 16px; border-radius: 6px; overflow-x: auto; }
    pre code { background: none; color: inherit; padding: 0; }
    .downloads { background: #eff6ff; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #2563eb; }
    .downloads h2 { margin-top: 0; color: #1e40af; }
    .version-section { margin: 15px 0; padding: 10px; background: white; border-radius: 4px; }
    .version-section h3 { margin: 0 0 10px 0; color: #1e3a8a; font-size: 1.1em; }
    .download-table { width: 100%; border-collapse: collapse; }
    .download-table td { padding: 6px 8px; }
    .platform-name { font-weight: 500; color: #1e3a8a; width: 200px; }
    .platform-links a { color: #2563eb; text-decoration: none; font-weight: 500; margin: 0 4px; }
    .platform-links a:hover { text-decoration: underline; }
  </style>
</head>
<body>
`)
}

func writeFooter(w io.Writer) {
	fmt.Fprint(w, `</body>
</html>
`)
}

func replaceInstallationSection(htmlContent []byte, downloadsHTML string) []byte {
	html := string(htmlContent)

	// Find the Installation section and replace it
	installStart := strings.Index(html, `<h2 id="install">`)
	if installStart == -1 {
		// Try alternate heading ID
		installStart = strings.Index(html, `<h2 id="installation">`)
	}
	if installStart == -1 {
		return htmlContent // Section not found, return as-is
	}

	// Find the next h2 tag
	nextH2 := strings.Index(html[installStart+20:], `<h2 id="`)
	if nextH2 == -1 {
		return htmlContent // Next section not found, return as-is
	}

	// Calculate actual position
	nextH2 += installStart + 20

	// Build replacement with the downloads table
	replacement := `<h2 id="installation">Installation</h2>

` + downloadsHTML + `
<p>Extract the archive and move the binary to your PATH:</p>

<pre><code class="language-bash"># macOS / Linux
tar -xzf kvx_*.tar.gz
sudo mv kvx /usr/local/bin/

# Windows
# Extract the .zip file and add kvx.exe to your PATH
</code></pre>

`

	// Replace the section
	result := html[:installStart] + replacement + html[nextH2:]
	return []byte(result)
}
