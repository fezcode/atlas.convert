package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

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

	flag.StringVar(&config.Source, "s", "", "Source media path")
	flag.StringVar(&config.Source, "source", "", "Source media path")
	flag.StringVar(&config.Destination, "d", "", "Destination path")
	flag.StringVar(&config.Destination, "destination", "", "Destination path")
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
		err := convert(config)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Println("atlas.convert - Image conversion tool")
	fmt.Println("\nUsage:")
	fmt.Println("  atlas.convert [flags]")
	fmt.Println("\nFlags:")
	fmt.Println("  -h, --help          Print help text")
	fmt.Println("  -v, --version       Print version")
	fmt.Println("  -i, --interactive   Launch the interactive version")
	fmt.Println("  -s, --source        Source media path")
	fmt.Println("  -d, --destination   Destination path")
	fmt.Println("  -f, --from          Source codec (jpeg, png, heic)")
	fmt.Println("  -t, --to            Destination codec (jpeg, png, heic)")
}

func runInteractive(config *Config) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Source path: ")
	config.Source, _ = reader.ReadString('\n')
	config.Source = strings.TrimSpace(config.Source)

	fmt.Print("Destination path: ")
	config.Destination, _ = reader.ReadString('\n')
	config.Destination = strings.TrimSpace(config.Destination)

	fmt.Print("From (jpeg/png/heic): ")
	config.From, _ = reader.ReadString('\n')
	config.From = strings.TrimSpace(config.From)

	fmt.Print("To (jpeg/png/heic): ")
	config.To, _ = reader.ReadString('\n')
	config.To = strings.TrimSpace(config.To)

	err := convert(*config)
	if err != nil {
		fmt.Printf("Error during conversion: %v\n", err)
	} else {
		fmt.Println("Conversion successful!")
	}
}

func convert(config Config) error {
	srcFile, err := os.Open(config.Source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Using image.Decode which uses registered decoders (including gen2brain/heic)
	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	destFile, err := os.Create(config.Destination)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	to := strings.ToLower(config.To)
	switch to {
	case "jpeg", "jpg":
		err = jpeg.Encode(destFile, img, nil)
	case "png":
		err = png.Encode(destFile, img)
	case "heic":
		return fmt.Errorf("encoding to HEIC is not supported yet")
	default:
		return fmt.Errorf("unsupported destination format: %s", to)
	}

	if err != nil {
		return fmt.Errorf("failed to encode %s: %w", to, err)
	}

	return nil
}
