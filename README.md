# font-manager

A command-line utility for managing fonts on Linux and macOS. Designed for both manual and automated workflows.

## Font sources
- Nerd fonts
- fontsource (includes google fonts)

## Installation

### Quick Install (Recommended)

Install with the installation script:
```bash
curl -sSL https://raw.githubusercontent.com/logandonley/font-manager/main/install.sh | bash
```

### Manual Installation

1. Download the latest binary for your system from the releases page
2. (Optional, but recommended): Rename the file to fm: `mv <Downloaded file> fm`
3. Make the binary executable: `chmod +x ./fm`
4. Move it to your PATH: `sudo mv ./fm /usr/local/bin/`

## Usage

Basic command structure:

```shell
fm [command] [options]
```

### Common use cases

Download a single font

```shell
fm install ComicShannsMono
```

Install multiple fonts

```shell
fm install ComicShannsMono Inter Rubik
```
