package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/gen2brain/heic"
)

var Version = "dev"

// ─── ANSI Style Helpers ───────────────────────────────────────────────────────

const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[37m"
)

func c(style, text string) string { return style + text + reset }
func cb(text string) string       { return bold + text + reset }
func cdim(text string) string     { return dim + text + reset }

func errMsg(msg string) {
	fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", c(bold+red, "ERROR"), msg)
}

func errHint(msg, hint string) {
	fmt.Fprintf(os.Stderr, "\n  %s %s\n", c(bold+red, "ERROR"), msg)
	fmt.Fprintf(os.Stderr, "  %s %s\n\n", c(dim, "hint:"), hint)
}


// ─── Supported Formats ───────────────────────────────────────────────────────

var readFormats = map[string]bool{
	"jpeg": true, "jpg": true, "png": true, "heic": true, "heif": true,
}
var writeFormats = map[string]bool{
	"jpeg": true, "jpg": true, "png": true,
}

func normalizeFormat(f string) string {
	f = strings.ToLower(strings.TrimSpace(f))
	if f == "jpg" {
		return "jpeg"
	}
	if f == "heif" {
		return "heic"
	}
	return f
}

func extToFormat(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	return normalizeFormat(ext)
}

func formatToExt(format string) string {
	switch format {
	case "jpeg":
		return "jpg"
	default:
		return format
	}
}

func formatList(m map[string]bool) string {
	seen := map[string]bool{}
	var out []string
	for k := range m {
		norm := normalizeFormat(k)
		if !seen[norm] {
			seen[norm] = true
			out = append(out, norm)
		}
	}
	return strings.Join(out, ", ")
}

// ─── Config ──────────────────────────────────────────────────────────────────

type Config struct {
	Source      string
	Destination string
	From        string
	To          string
	Interactive bool
}

// ─── Main ────────────────────────────────────────────────────────────────────

func main() {
	config := Config{}

	flag.StringVar(&config.Source, "s", "", "")
	flag.StringVar(&config.Source, "source", "", "")
	flag.StringVar(&config.Destination, "d", "", "")
	flag.StringVar(&config.Destination, "destination", "", "")
	flag.StringVar(&config.From, "f", "", "")
	flag.StringVar(&config.From, "from", "", "")
	flag.StringVar(&config.To, "t", "", "")
	flag.StringVar(&config.To, "to", "", "")
	flag.BoolVar(&config.Interactive, "i", false, "")
	flag.BoolVar(&config.Interactive, "interactive", false, "")

	help := flag.Bool("h", false, "")
	flag.BoolVar(help, "help", false, "")
	version := flag.Bool("v", false, "")
	flag.BoolVar(version, "version", false, "")

	flag.Usage = func() { printHelp() }
	flag.Parse()

	if *version {
		fmt.Printf("%s %s\n", c(bold+cyan, "atlas.convert"), c(dim, "v"+Version))
		return
	}

	if *help {
		printHelp()
		return
	}

	// No args at all -> show help
	if len(os.Args) == 1 {
		printHelp()
		return
	}

	if config.Interactive {
		runInteractive(&config)
		return
	}

	// ── Detect shell-expanded globs ──
	// When the user writes: atlas.convert -s *.png -d out/ -t jpeg
	// the shell expands *.png BEFORE we see it, producing:
	//   atlas.convert -s a.png b.png c.png -d out/ -t jpeg
	// Go's flag parser captures -s a.png, then stops at b.png (non-flag),
	// so -d and -t never get parsed. Detect this and give a clear message.
	leftover := flag.Args()
	if len(leftover) > 0 {
		hasUnparsedFlags := false
		for _, arg := range leftover {
			if strings.HasPrefix(arg, "-") {
				hasUnparsedFlags = true
				break
			}
		}
		if hasUnparsedFlags {
			errHint("it looks like your shell expanded a glob pattern before atlas.convert could read it.",
				"quote your glob so the shell passes it through: -s \"*.png\" instead of -s *.png")
			os.Exit(1)
		}
	}

	// ── Validate non-interactive args ──

	if config.Source == "" {
		errHint("missing source path.", "use -s <path> or -s \"*.heic\"")
		os.Exit(1)
	}
	if config.Destination == "" {
		errHint("missing destination path.", "use -d <path> or -d \"output/\"")
		os.Exit(1)
	}
	if config.To == "" {
		errHint("missing target format.", "use -t <format>  (supported: "+formatList(writeFormats)+")")
		os.Exit(1)
	}

	config.From = normalizeFormat(config.From)
	config.To = normalizeFormat(config.To)

	if config.To != "" && !writeFormats[config.To] {
		errHint(fmt.Sprintf("cannot encode to %q.", config.To),
			"supported output formats: "+formatList(writeFormats))
		os.Exit(1)
	}
	if config.From != "" && !readFormats[config.From] {
		errHint(fmt.Sprintf("unknown source format %q.", config.From),
			"supported input formats: "+formatList(readFormats))
		os.Exit(1)
	}

	// If leftover positional args remain (and we passed the flag check above),
	// they're bare file paths from shell glob expansion where all flags were
	// placed before the glob. e.g.: atlas.convert -d out/ -t jpeg -s *.png
	// In this case -s gets the first file, the rest land in leftover.
	runBatch(config, leftover)
}

// ─── Help ────────────────────────────────────────────────────────────────────

func printHelp() {
	fmt.Println()
	fmt.Printf("  %s %s\n", c(bold+cyan, "atlas.convert"), c(dim, "v"+Version))
	fmt.Printf("  %s\n", c(dim, "Image conversion tool - part of the Atlas Suite"))
	fmt.Println()

	fmt.Printf("  %s\n", cb("USAGE"))
	fmt.Printf("    %s [flags]\n", c(cyan, "atlas.convert"))
	fmt.Println()

	fmt.Printf("  %s\n", cb("FLAGS"))
	fmt.Printf("    %s    Source path (file, directory, or glob)       %s\n", c(green, "-s, --source <path>"), c(dim, "required"))
	fmt.Printf("    %s    Destination path (directory or pattern)      %s\n", c(green, "-d, --dest <path>  "), c(dim, "required"))
	fmt.Printf("    %s    Target format: jpeg, png                     %s\n", c(green, "-t, --to <format>  "), c(dim, "required"))
	fmt.Printf("    %s    Source format: jpeg, png, heic               %s\n", c(green, "-f, --from <format>"), c(dim, "auto-detected"))
	fmt.Printf("    %s    Launch interactive guided mode\n", c(green, "-i, --interactive  "))
	fmt.Printf("    %s    Show this help\n", c(green, "-h, --help         "))
	fmt.Printf("    %s    Print version\n", c(green, "-v, --version      "))
	fmt.Println()

	fmt.Printf("  %s\n", cb("PLACEHOLDERS"))
	fmt.Printf("    Use these tokens in the destination path for dynamic filenames:\n")
	fmt.Printf("    %s   original filename without extension\n", c(yellow, "{name}"))
	fmt.Printf("    %s    sequence index (0, 1, 2, ...)\n", c(yellow, "{idx}"))
	fmt.Printf("    %s   current date (YYYY-MM-DD)\n", c(yellow, "{date}"))
	fmt.Println()

	fmt.Printf("  %s\n", cb("EXAMPLES"))
	fmt.Println()
	fmt.Printf("    %s\n", cdim("# Convert a single HEIC photo to JPEG"))
	fmt.Printf("    atlas.convert %s photo.heic %s photo.jpg %s jpeg\n",
		c(green, "-s"), c(green, "-d"), c(green, "-t"))
	fmt.Println()
	fmt.Printf("    %s\n", cdim("# Batch convert all HEIC files in current directory to PNG"))
	fmt.Printf("    atlas.convert %s \"*.heic\" %s converted/ %s png\n",
		c(green, "-s"), c(green, "-d"), c(green, "-t"))
	fmt.Println()
	fmt.Printf("    %s\n", cdim("# Convert PNGs to JPEGs with custom naming"))
	fmt.Printf("    atlas.convert %s \"raw/*.png\" %s \"web/{name}_optimized.jpg\" %s jpeg\n",
		c(green, "-s"), c(green, "-d"), c(green, "-t"))
	fmt.Println()
	fmt.Printf("    %s\n", cdim("# Convert a directory of JPGs to numbered PNGs"))
	fmt.Printf("    atlas.convert %s photos/ %s \"archive/IMG_{idx}.png\" %s jpg %s png\n",
		c(green, "-s"), c(green, "-d"), c(green, "-f"), c(green, "-t"))
	fmt.Println()
	fmt.Printf("    %s\n", cdim("# Interactive guided mode"))
	fmt.Printf("    atlas.convert %s\n", c(green, "-i"))
	fmt.Println()

	fmt.Printf("  %s\n", cb("FORMAT SUPPORT"))
	fmt.Printf("    %-8s  %s  %s\n", "", c(dim, "read"), c(dim, "write"))
	fmt.Printf("    %-8s  %s    %s\n", "JPEG", c(green, " yes"), c(green, " yes"))
	fmt.Printf("    %-8s  %s    %s\n", "PNG", c(green, " yes"), c(green, " yes"))
	fmt.Printf("    %-8s  %s    %s\n", "HEIC", c(green, " yes"), c(dim+yellow, "  - "))
	fmt.Println()
}

// ─── Interactive Mode ────────────────────────────────────────────────────────

func runInteractive(config *Config) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("  %s %s\n", c(bold+cyan, "atlas.convert"), c(dim, "interactive mode"))
	fmt.Printf("  %s\n", c(dim, "Answer the prompts below to convert your images."))
	fmt.Println()

	// Source
	for {
		fmt.Printf("  %s %s\n", c(bold+white, "Source path"), c(dim, "(file, directory, or glob like \"*.heic\")"))
		fmt.Printf("  %s ", c(cyan, ">"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Printf("  %s\n\n", c(red, "Please enter a source path."))
			continue
		}
		config.Source = input
		break
	}
	fmt.Println()

	// Source format (optional)
	fmt.Printf("  %s %s\n", c(bold+white, "Source format"), c(dim, "(jpeg, png, heic) - leave blank to auto-detect"))
	fmt.Printf("  %s ", c(cyan, ">"))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input != "" {
		f := normalizeFormat(input)
		if !readFormats[f] {
			fmt.Printf("  %s Unknown format %q. Supported: %s\n", c(yellow, "warn:"), input, formatList(readFormats))
		} else {
			config.From = f
		}
	}
	fmt.Println()

	// Destination
	for {
		fmt.Printf("  %s %s\n", c(bold+white, "Destination path"), c(dim, "(directory or pattern with {name}, {idx}, {date})"))
		fmt.Printf("  %s ", c(cyan, ">"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Printf("  %s\n\n", c(red, "Please enter a destination path."))
			continue
		}
		config.Destination = input
		break
	}
	fmt.Println()

	// Target format
	for {
		fmt.Printf("  %s %s\n", c(bold+white, "Target format"), c(dim, "(jpeg, png)"))
		fmt.Printf("  %s ", c(cyan, ">"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Printf("  %s\n\n", c(red, "Please enter a target format."))
			continue
		}
		f := normalizeFormat(input)
		if !writeFormats[f] {
			fmt.Printf("  %s Cannot write to %q. Supported: %s\n\n", c(red, "error:"), input, formatList(writeFormats))
			continue
		}
		config.To = f
		break
	}
	fmt.Println()

	// Preview
	files, err := resolveSourceFiles(config.Source, config.From)
	if err != nil {
		errMsg(fmt.Sprintf("could not resolve source files: %v", err))
		os.Exit(1)
	}
	if len(files) == 0 {
		errHint("no matching files found.", "check your source path and format filter")
		os.Exit(1)
	}

	// Validate destination for multi-file sources
	if len(files) > 1 {
		if err := validateMultiFileDest(config.Destination); err != nil {
			errHint(err.Error(),
				"use a directory path (e.g. output/) or a pattern with placeholders (e.g. \"out/{name}.png\")")
			os.Exit(1)
		}
	}

	fmt.Printf("  %s\n", cb("PREVIEW"))
	toExt := formatToExt(config.To)
	maxPreview := 8
	for i, f := range files {
		if i >= maxPreview {
			fmt.Printf("    %s\n", c(dim, fmt.Sprintf("... and %d more files", len(files)-maxPreview)))
			break
		}
		dest := resolveDestPath(f, config.Destination, toExt, i)
		fmt.Printf("    %s %s %s\n", c(dim, filepath.Base(f)), c(cyan, "->"), c(green, dest))
	}
	fmt.Println()

	fmt.Printf("  %s %s\n", c(bold+white, "Proceed?"), c(dim, "(Y/n)"))
	fmt.Printf("  %s ", c(cyan, ">"))
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "" && confirm != "y" && confirm != "yes" {
		fmt.Printf("\n  %s\n\n", c(dim, "Cancelled."))
		return
	}
	fmt.Println()

	runConversion(files, config.Destination, config.To)
}

// ─── Batch Mode ──────────────────────────────────────────────────────────────

func runBatch(config Config, extraSourceFiles []string) {
	files, err := resolveSourceFiles(config.Source, config.From)
	if err != nil {
		errMsg(fmt.Sprintf("could not resolve source files: %v", err))
		os.Exit(1)
	}

	// Merge any extra positional args from shell glob expansion.
	// These are bare file paths the shell expanded before we saw them.
	for _, extra := range extraSourceFiles {
		info, err := os.Stat(extra)
		if err != nil || info.IsDir() {
			continue
		}
		ext := extToFormat(filepath.Ext(extra))
		if !readFormats[ext] {
			continue
		}
		if config.From != "" && ext != config.From {
			continue
		}
		// Avoid duplicates (the -s value may already be in the list)
		dup := false
		for _, f := range files {
			if f == extra {
				dup = true
				break
			}
		}
		if !dup {
			files = append(files, extra)
		}
	}

	if len(files) == 0 {
		if config.From != "" {
			errHint("no matching files found.",
				fmt.Sprintf("no %s files matched %q - check the path or try without -f to include all formats", config.From, config.Source))
		} else {
			errHint("no matching files found.",
				fmt.Sprintf("nothing matched %q - check the path exists", config.Source))
		}
		os.Exit(1)
	}

	// Validate destination makes sense for the number of files
	if len(files) > 1 {
		if err := validateMultiFileDest(config.Destination); err != nil {
			errHint(err.Error(),
				"use a directory path (e.g. -d output/) or a pattern with placeholders (e.g. -d \"out/{name}.png\")")
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Printf("  %s converting %s file(s) to %s\n\n",
		c(bold+cyan, "atlas.convert"),
		c(bold+white, fmt.Sprintf("%d", len(files))),
		c(bold+green, strings.ToUpper(config.To)))

	runConversion(files, config.Destination, config.To)
}

// validateMultiFileDest checks that a destination can handle multiple files
// without silently overwriting the same path.
func validateMultiFileDest(dest string) error {
	// Directory-style paths are fine
	if strings.HasSuffix(dest, "/") || strings.HasSuffix(dest, "\\") {
		return nil
	}

	// Existing directory is fine
	info, err := os.Stat(dest)
	if err == nil && info.IsDir() {
		return nil
	}

	// Patterns with per-file placeholders are fine
	if strings.Contains(dest, "{name}") || strings.Contains(dest, "{idx}") {
		return nil
	}

	// Anything else is a single file path — multiple files would overwrite it
	return fmt.Errorf("destination %q is a single file, but source matches multiple files.", dest)
}

// ─── Conversion Engine ──────────────────────────────────────────────────────

func runConversion(files []string, destPattern, toFormat string) {
	successCount := 0
	total := len(files)
	toExt := formatToExt(toFormat)
	startTime := time.Now()

	for i, srcPath := range files {
		destPath := resolveDestPath(srcPath, destPattern, toExt, i)

		// Create parent directory
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("  %s  %s  %s\n",
				c(dim, fmt.Sprintf("[%d/%d]", i+1, total)),
				c(bold+red, "FAIL"),
				c(red, fmt.Sprintf("cannot create directory %s: %v", destDir, err)))
			continue
		}

		err := convertSingle(srcPath, destPath, toFormat)
		if err != nil {
			fmt.Printf("  %s  %s  %s %s\n",
				c(dim, fmt.Sprintf("[%d/%d]", i+1, total)),
				c(bold+red, "FAIL"),
				filepath.Base(srcPath),
				c(dim+red, err.Error()))
		} else {
			absDest, _ := filepath.Abs(destPath)
			fmt.Printf("  %s  %s  %s\n",
				c(dim, fmt.Sprintf("[%d/%d]", i+1, total)),
				c(green, " ok "),
				absDest)
			successCount++
		}
	}

	elapsed := time.Since(startTime)

	fmt.Println()
	if successCount == total {
		fmt.Printf("  %s %d file(s) converted in %s\n\n",
			c(bold+green, "Done!"), successCount, formatDuration(elapsed))
	} else {
		fmt.Printf("  %s %d/%d succeeded, %s %d failed %s\n\n",
			c(bold+green, "Done!"), successCount, total,
			c(bold+red, ""), total-successCount, c(dim, "in "+formatDuration(elapsed)))
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// ─── File Resolution ────────────────────────────────────────────────────────

func resolveSourceFiles(source, fromFormat string) ([]string, error) {
	var files []string

	// Check if it's a directory
	info, err := os.Stat(source)
	if err == nil && info.IsDir() {
		entries, err := os.ReadDir(source)
		if err != nil {
			return nil, fmt.Errorf("cannot read directory %q: %w", source, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := extToFormat(filepath.Ext(entry.Name()))
			if !readFormats[ext] {
				continue // skip non-image files
			}
			if fromFormat != "" && ext != fromFormat {
				continue // skip files not matching the filter
			}
			files = append(files, filepath.Join(source, entry.Name()))
		}
		return files, nil
	}

	// Try as a glob
	matches, err := filepath.Glob(source)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %q: %w", source, err)
	}

	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		ext := extToFormat(filepath.Ext(m))
		if !readFormats[ext] {
			continue
		}
		if fromFormat != "" && ext != fromFormat {
			continue
		}
		files = append(files, m)
	}

	return files, nil
}

func resolveDestPath(srcPath, destPattern, toExt string, index int) string {
	// If destPattern ends with separator, treat as directory
	if strings.HasSuffix(destPattern, "/") || strings.HasSuffix(destPattern, "\\") {
		name := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		return filepath.Join(destPattern, name+"."+toExt)
	}

	// If destPattern is an existing directory
	info, err := os.Stat(destPattern)
	if err == nil && info.IsDir() {
		name := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		return filepath.Join(destPattern, name+"."+toExt)
	}

	// Apply placeholders
	name := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	res := strings.ReplaceAll(destPattern, "{name}", name)
	res = strings.ReplaceAll(res, "{idx}", fmt.Sprintf("%d", index))
	res = strings.ReplaceAll(res, "{date}", time.Now().Format("2006-01-02"))

	// Ensure extension
	if filepath.Ext(res) == "" {
		res += "." + toExt
	}

	return res
}

// ─── Image Conversion ───────────────────────────────────────────────────────

func convertSingle(srcPath, destPath, toFormat string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("cannot open: %v", err)
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("decode error: %v", err)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("cannot create: %v", err)
	}
	defer destFile.Close()

	switch toFormat {
	case "jpeg":
		return jpeg.Encode(destFile, img, &jpeg.Options{Quality: 92})
	case "png":
		return png.Encode(destFile, img)
	default:
		return fmt.Errorf("unsupported target format: %s", toFormat)
	}
}
