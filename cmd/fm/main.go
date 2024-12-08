package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/logandonley/font-manager/pkg/fm"
	"github.com/spf13/cobra"
)

var manager *fm.DefaultManager

func main() {
	var err error
	manager, err = fm.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing font manager: %v\n", err)
		os.Exit(1)
	}

	// Register default sources
	if err := manager.RegisterSource(fm.NewNerdFontsSource()); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering NerdFonts source: %v\n", err)
		os.Exit(1)
	}
	if err := manager.RegisterSource(fm.NewFontSourceAPI()); err != nil {
		fmt.Fprintf(os.Stderr, "Error registering FontSource API: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "fm",
	Short: "fm is a font manager for Linux and macOS",
	Long: `A font manager that supports multiple sources including:
- Nerd Fonts
- FontSource
- Direct URLs

Examples:
  # Install a font from any source
  fm install "FiraCode"

  # Install specifically from NerdFonts
  fm install "FiraCode@nerdfonts"

  # Install from a direct URL
  fm install https://example.com/font.zip

  # Install multiple fonts from a config file
  fm install -f fonts.txt`,
}

var installCmd = &cobra.Command{
	Use:   "install [font names...] | -f <file>",
	Short: "Install one or more fonts",
	Long: `Install one or more fonts from any supported source.
You can specify multiple fonts and mix sources:

Examples:
  # Install a single font
  fm install "FiraCode"

  # Install multiple fonts
  fm install "FiraCode" "RobotoMono" "JetBrainsMono"

  # Install fonts from specific sources
  fm install "FiraCode@nerdfonts" "RobotoMono@fontsource"

  # Install from URLs and sources together
  fm install "FiraCode@nerdfonts" https://example.com/font.zip

  # Install multiple fonts from a config file
  fm install -f fonts.txt`,
	Args: func(cmd *cobra.Command, args []string) error {
		fileFlag, _ := cmd.Flags().GetString("file")
		if fileFlag != "" {
			if len(args) > 0 {
				return fmt.Errorf("when using -f flag, no additional arguments should be provided")
			}
			return nil
		}
		if len(args) < 1 {
			return fmt.Errorf("requires at least 1 font name when not using -f flag")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, _ := cmd.Flags().GetString("file")
		if configFile != "" {
			file, err := os.Open(configFile)
			if err != nil {
				return fmt.Errorf("opening config file: %w", err)
			}
			defer file.Close()

			fmt.Printf("Installing fonts from %s...\n", configFile)
			if err := manager.InstallFromConfig(cmd.Context(), file); err != nil {
				return fmt.Errorf("installing fonts from config: %w", err)
			}
			fmt.Println("Successfully installed fonts from config file")
			return nil
		}

		// Track installation results
		var failed []string
		var skipped []string
		successful := 0

		// Install each font specified
		for _, name := range args {
			fmt.Printf("Installing %s...\n", name)
			if err := manager.Install(cmd.Context(), name); err != nil {
				if strings.Contains(err.Error(), "already installed") {
					fmt.Printf("Skipped %s (already installed)\n", name)
					skipped = append(skipped, name)
					continue
				}
				fmt.Fprintf(os.Stderr, "Error installing %s: %v\n", name, err)
				failed = append(failed, name)
				continue
			}
			fmt.Printf("Successfully installed %s\n", name)
			successful++
		}

		// Print summary
		fmt.Printf("\nInstallation Summary:\n")
		fmt.Printf("Successfully installed: %d\n", successful)
		if len(skipped) > 0 {
			fmt.Printf("Skipped (already installed): %d\n", len(skipped))
			for _, name := range skipped {
				fmt.Printf("  - %s\n", name)
			}
		}
		if len(failed) > 0 {
			fmt.Printf("Failed to install: %d\n", len(failed))
			fmt.Println("Failed fonts:")
			for _, name := range failed {
				fmt.Printf("  - %s\n", name)
			}
			return fmt.Errorf("some fonts failed to install")
		}

		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [font name]",
	Short: "Uninstall a font",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fmt.Printf("Uninstalling %s...\n", name)
		if err := manager.Uninstall(cmd.Context(), name); err != nil {
			return fmt.Errorf("uninstalling %s: %w", name, err)
		}
		fmt.Printf("Successfully uninstalled %s\n", name)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed fonts",
	RunE: func(cmd *cobra.Command, args []string) error {
		fonts, err := manager.List(cmd.Context())
		if err != nil {
			return fmt.Errorf("listing fonts: %w", err)
		}

		if len(fonts) == 0 {
			fmt.Println("No fonts installed")
			return nil
		}

		fmt.Println("Installed fonts:")
		for _, font := range fonts {
			if font.Source != "" {
				fmt.Printf("  - %s (from %s)\n", font.Name, font.Source)
			} else {
				fmt.Printf("  - %s\n", font.Name)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(listCmd)

	installCmd.Flags().StringP("file", "f", "", "Install fonts from a config file")
}
