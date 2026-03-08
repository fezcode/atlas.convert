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

type Config struct {
	Source      string
	Destination string
	From        string
	To          string
	Interactive bool
}

func main() {
	config := Config{}

	flag.StringVar(&config.Source, "s", "", "Source media path (supports globs like *.heic)")
	flag.StringVar(&config.Source, "source", "", "Source media path (supports globs like *.heic)")
	flag.StringVar(&config.Destination, "d", "", "Destination path (supports placeholders: {name}, {idx}, {date})")
	flag.StringVar(&config.Destination, "destination", "", "Destination path (supports placeholders: {name}, {idx}, {date})")
	flag.StringVar(&config.From, "f", "", "Source codec (jpeg, png, heic)")
	flag.StringVar(&config.From, "from", "", "Source codec (jpeg, png, heic)")
	flag.StringVar(&config.To, "t", "", "Destination codec (jpeg, png, heic)")
	flag.StringVar(&config.To, "to", "", "Destination codec (jpeg, png, heic)")
	flag.BoolVar(&config.Interactive, "i", false, "Launch interactive version")
	flag.BoolVar(&config.Interactive, "interactive", false, "Launch interactive version")

	help := flag.Bool("h", false, "Print help text")
	flag.BoolVar(help, "help", false, "Print help text")
	version := flag.Bool("v", false, "Print version")
	flag.BoolVar(version, "version", false, "Print version")

	flag.Parse()

	if *version {
		fmt.Printf("atlas.convert v%s\n", Version)
		return
	}

	if *help || (len(os.Args) == 1 && !config.Interactive) {
		printHelp()
		return
	}

	if config.Interactive {
		runInteractive(&config)
	} else {
		if config.Source == "" || config.Destination == "" || config.From == "" || config.To == "" {
			fmt.Println("Error: Missing required arguments for non-interactive mode.")
			printHelp()
			os.Exit(1)
		}
		runBatch(config)
	}
}

func printHelp() {
	fmt.Println("atlas.convert - Image conversion tool")
	fmt.Println("\nUsage:")
	fmt.Println("  atlas.convert [flags]")
	fmt.Println("\nBatch & Glob Examples:")
	fmt.Println("  atlas.convert -s \"*.heic\" -d \"outputs/\" -t png")
	fmt.Println("  atlas.convert -s \"raw/*.png\" -d \"processed/{name}_web.jpg\" -t jpeg")
	fmt.Println("  atlas.convert -s \"images/\" -d \"out/{idx}_img.png\" -f jpg -t png")
	fmt.Println("\nPlaceholders:")
	fmt.Println("  {name}   Original filename without extension")
	fmt.Println("  {idx}    Sequence index (0, 1, 2...)")
	fmt.Println("  {date}   Current date (YYYY-MM-DD)")
	fmt.Println("\nFlags:")
	fmt.Println("  -h, --help          Print help text")
	fmt.Println("  -v, --version       Print version")
	fmt.Println("  -i, --interactive   Launch the interactive version")
	flag.PrintDefaults()
}

func runInteractive(config *Config) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Source path (can be glob or dir): ")
	config.Source, _ = reader.ReadString('\n')
	config.Source = strings.TrimSpace(config.Source)

	fmt.Print("Destination path (can be dir or pattern): ")
	config.Destination, _ = reader.ReadString('\n')
	config.Destination = strings.TrimSpace(config.Destination)

	fmt.Print("From (jpeg/png/heic): ")
	config.From, _ = reader.ReadString('\n')
	config.From = strings.TrimSpace(config.From)

	fmt.Print("To (jpeg/png/heic): ")
	config.To, _ = reader.ReadString('\n')
	config.To = strings.TrimSpace(config.To)

	runBatch(*config)
}

func runBatch(config Config) {
	files, err := resolveSourceFiles(config.Source, config.From)
	if err != nil {
		fmt.Printf("Error resolving source files: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No matching source files found.")
		return
	}

	fmt.Printf("Processing %d files...\n", len(files))

	successCount := 0
	for i, srcPath := range files {
		destPath := resolveDestPath(srcPath, config.Destination, config.To, i)

		// Create parent directory if it doesn't exist
		destDir := filepath.Dir(destPath)
		if _, err := os.Stat(destDir); os.IsNotExist(err) {
			os.MkdirAll(destDir, 0755)
		}

		err := convertSingle(srcPath, destPath, config.To)
		if err != nil {
			fmt.Printf("[%d/%d] ❌ Failed %s: %v\n", i+1, len(files), srcPath, err)
		} else {
			absDest, _ := filepath.Abs(destPath)
			fmt.Printf("[%d/%d] ✅ Converted: %s\n", i+1, len(files), absDest)
			successCount++
		}
	}

	fmt.Printf("\nDone! Successfully converted %d/%d files.\n", successCount, len(files))
}

func resolveSourceFiles(source, from string) ([]string, error) {
	// Check if it's a directory
	info, err := os.Stat(source)
	if err == nil && info.IsDir() {
		ext := "." + strings.ToLower(from)
		if from == "jpeg" {
			ext = ".jpg" // Standardize
		}
		
		var files []string
		entries, _ := os.ReadDir(source)
		for _, entry := range entries {
			if !entry.IsDir() && (strings.ToLower(filepath.Ext(entry.Name())) == ext || (from == "jpeg" && strings.ToLower(filepath.Ext(entry.Name())) == ".jpeg")) {
				files = append(files, filepath.Join(source, entry.Name()))
			}
		}
		return files, nil
	}

	// Try glob
	return filepath.Glob(source)
}

func resolveDestPath(srcPath, destPattern, toExt string, index int) string {
	// If destPattern is an existing directory or ends in a slash, treat it as a folder
	if strings.HasSuffix(destPattern, "/") || strings.HasSuffix(destPattern, "\\") {
		name := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
		return filepath.Join(destPattern, name+"."+toExt)
	}

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

	// Ensure extension if not provided in pattern
	if filepath.Ext(res) == "" {
		res += "." + toExt
	}

	return res
}

func convertSingle(srcPath, destPath, toExt string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	switch strings.ToLower(toExt) {
	case "jpeg", "jpg":
		return jpeg.Encode(destFile, img, nil)
	case "png":
		return png.Encode(destFile, img)
	default:
		return fmt.Errorf("unsupported destination format: %s", toExt)
	}
}
