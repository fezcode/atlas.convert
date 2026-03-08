# Atlas Convert

![Banner Image](./banner-image.png)

> [!IMPORTANT]
> **atlas.convert** is part of the **Atlas Suite**—a collection of high-visibility, local-first terminal utilities designed for power users who demand precision and aesthetic clarity.

**atlas.convert** is a high-performance image conversion tool that simplifies media workflows directly from your terminal. Whether you're batch-processing assets for a project or quickly converting a single HEIC photo from your phone, Atlas Convert provides a seamless, dependency-free experience with the signature Atlas aesthetic.

![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Build System](https://img.shields.io/badge/build-gobake-gold)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)

## ✨ Key Features

- 🖼️ **Format Support:** Seamlessly convert between **JPEG**, **PNG**, and **HEIC** (Decoding).
- 🕹️ **Interactive Mode:** A guided, step-by-step TUI prompt for users who prefer not to memorize flags.
- ⚡ **Blazing Fast:** Written in Go with native imaging libraries for near-instantaneous conversion.
- 🍎 **HEIC Support:** Includes built-in decoding for modern iPhone/macOS image formats without needing external system libraries.
- 🎨 **Aesthetic CLI:** Clean, high-contrast output consistent with the Atlas Suite design language.
- 📦 **Zero Dependencies:** Compiles to a single, static binary—no `imagemagick` or `ffmpeg` required.

## 🚀 Installation

### Using Gobake (Recommended)
This project uses the Atlas **gobake** orchestration tool for cross-platform builds.

```bash
git clone https://github.com/fezcode/atlas.convert
cd atlas.convert
gobake build
```
The compiled binary will be located in the `build/` directory.

### From Source
```bash
go build -o atlas.convert .
```

## ⌨️ Usage

### Quick CLI Conversion
Convert a HEIC image to a web-ready PNG:
```bash
atlas.convert -s photo.heic -d photo.png -f heic -t png
```

### Interactive Mode
Launch the guided conversion wizard:
```bash
atlas.convert -i
```

### Options & Flags

| Flag | Shorthand | Description |
|------|-----------|-------------|
| `--source` | `-s` | Path to the source media file |
| `--destination` | `-d` | Path where the converted file will be saved |
| `--from` | `-f` | Source codec/format (jpeg, png, heic) |
| `--to` | `-t` | Destination codec/format (jpeg, png) |
| `--interactive` | `-i` | Launch the interactive guided mode |
| `--help` | `-h` | Print help text and usage examples |
| `--version` | `-v` | Print current version |

## 🛠️ Supported Formats

| Format | Read (Decode) | Write (Encode) |
|--------|:-------------:|:--------------:|
| **JPEG** | ✅ | ✅ |
| **PNG** | ✅ | ✅ |
| **HEIC** | ✅ | ❌ (Coming soon) |

## 🏗️ Architecture & Philosophy

- **Local-First:** All processing happens on your machine. Your photos never touch the cloud.
- **Precision:** Uses high-quality resampling and encoding parameters to ensure no loss in visual fidelity during conversion.
- **Portability:** Built with pure Go (where possible) to ensure the tool works across all major operating systems with zero setup.

## 📄 License
MIT License - Copyright (c) 2026 FezCode.
