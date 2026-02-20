// Package main is the entry point for the pr-dashboard TUI application.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/github"
	"github.com/jgordijn/pr-dashboard/internal/tui"
)

// Version is the application version, set at build time.
var Version = "dev"

// Terminal size requirements per configuration/spec.md
const (
	minTerminalWidth  = 80
	minTerminalHeight = 24
)

func main() {
	os.Exit(run())
}

// run is the main entry point, returning an exit code.
// This design allows for easier testing and ensures proper cleanup.
func run() int {
	// Parse command line flags
	var (
		configPath  string
		showVersion bool
		showHelp    bool
	)

	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showHelp, "help", false, "Show usage information")
	flag.Parse()

	// Handle help flag
	if showHelp {
		printUsage()
		return 0
	}

	// Handle version flag
	if showVersion {
		fmt.Printf("pr-dashboard version %s\n", Version)
		return 0
	}

	// Validate TTY - per configuration/spec.md: "Interactive terminal required"
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "Error: Interactive terminal required. Run in a terminal window.")
		return 1
	}

	// Validate terminal size - per configuration/spec.md: minimum 80x24
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to determine terminal size: %v\n", err)
		return 1
	}
	if width < minTerminalWidth || height < minTerminalHeight {
		fmt.Fprintf(os.Stderr, "Error: Terminal too small (minimum %dx%d, current %dx%d)\n",
			minTerminalWidth, minTerminalHeight, width, height)
		return 1
	}

	// Check gh CLI installation - per configuration/spec.md
	if err := config.CheckGHCLI(); err != nil {
		if errors.Is(err, config.ErrGHCLINotFound) {
			fmt.Fprintln(os.Stderr, "Error: gh CLI not found. Install from https://cli.github.com")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	// Check gh CLI authentication - per configuration/spec.md
	if err := config.CheckGHAuth(); err != nil {
		if errors.Is(err, config.ErrGHNotAuthenticated) {
			fmt.Fprintln(os.Stderr, "Error: gh CLI not authenticated. Run `gh auth login` first")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}

	// Load configuration
	var cfg *config.Config
	if configPath != "" {
		// Load from custom path
		cfg, err = config.LoadFromPath(configPath)
	} else {
		// Load from default path
		cfg, err = config.Load()
	}

	if err != nil {
		// If config not found, run the setup wizard
		if errors.Is(err, config.ErrConfigNotFound) {
			cfg, err = config.RunWizard()
			if err != nil {
				if errors.Is(err, config.ErrWizardCancelled) {
					fmt.Fprintln(os.Stderr, "Setup cancelled.")
					return 1
				}
				fmt.Fprintf(os.Stderr, "Error: Setup failed: %v\n", err)
				return 1
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: Failed to load configuration: %v\n", err)
			return 1
		}
	}

	// Validate configuration - per configuration/spec.md
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Create GitHub client using the configured user's token.
	// go-gh uses whichever gh account is "active", which may differ from
	// the configured username. Fetching the token explicitly ensures we
	// always authenticate as the configured user.
	token, err := config.GHAuthToken(cfg.General.Username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get auth token for %s: %v\n", cfg.General.Username, err)
		return 1
	}
	client, err := github.NewClientWithToken(token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create GitHub client: %v\n", err)
		return 1
	}

	// Create TUI model
	model := tui.NewModel(cfg, client)

	// Create Bubble Tea program with alt screen (full screen mode)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		program.Kill()
	}()

	// Start the TUI
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

// printUsage prints the usage information.
func printUsage() {
	fmt.Println("pr-dashboard - A TUI for monitoring your GitHub pull requests")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  pr-dashboard [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --config <path>  Path to configuration file")
	fmt.Println("  --version        Show version information")
	fmt.Println("  --help           Show this help message")
	fmt.Println()
	fmt.Println("Key Bindings:")
	fmt.Println("  j/k or arrows    Navigate up/down")
	fmt.Println("  gg/G             Jump to top/bottom")
	fmt.Println("  o/O              Toggle organization collapse")
	fmt.Println("  d                Toggle draft visibility")
	fmt.Println("  c                Cycle display mode")
	fmt.Println("  w                Toggle watch mode")
	fmt.Println("  u                Update branch (when behind)")
	fmt.Println("  r                Refresh")
	fmt.Println("  Enter            Open PR in browser")
	fmt.Println("  ?                Show help")
	fmt.Println("  q/Esc            Quit")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Default: ~/.config/pr-dashboard/config.toml")
	fmt.Println("  Run without config to start the setup wizard.")
}
