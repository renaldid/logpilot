package cmd

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/renaldid/logpilot/internal/config"
	"github.com/renaldid/logpilot/internal/pipeline"
	"github.com/renaldid/logpilot/internal/source"
	"github.com/renaldid/logpilot/internal/tui"
)

var rootFlags struct {
	configFile string
	follow     bool
}

// programOptions are the options passed to tea.NewProgram; overridable in tests
// to avoid requiring a real TTY.
var programOptions = []tea.ProgramOption{tea.WithAltScreen()}

// tuiRunner starts the BubbleTea program; overridable in tests.
var tuiRunner = func(m tea.Model) error {
	_, err := tea.NewProgram(m, programOptions...).Run()
	return err
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "logpilot",
		Short: "All your service logs, one beautiful terminal",
		Long: `logpilot aggregates log streams from Docker Compose containers, log files,
and systemd units into a single interactive TUI — with real-time fuzzy search
and per-service filtering.`,
		RunE: runRoot,
	}

	root.PersistentFlags().StringVar(&rootFlags.configFile, "config", "", "config file (default: .logpilot.yaml in current dir)")
	root.PersistentFlags().BoolVar(&rootFlags.follow, "follow", true, "auto-scroll to the latest log entry")

	root.AddCommand(newVersionCmd())

	return root
}

func runRoot(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(rootFlags.configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cmd.Flags().Changed("follow") {
		cfg.Follow = rootFlags.follow
	}

	sources, err := buildSources(cfg)
	if err != nil {
		return err
	}

	buf := pipeline.NewRingBuffer(cfg.BufferSize)
	agg := pipeline.NewAggregator(sources, buf, cfg.BufferSize)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entryCh, err := agg.Start(ctx)
	if err != nil {
		return fmt.Errorf("start aggregator: %w", err)
	}

	state := tui.NewState(cfg.Follow, cfg.Colors)
	model := tui.New(entryCh, state)

	return tuiRunner(model)
}

// buildSources constructs LogSource instances from the config.
// Systemd sources are skipped on unsupported platforms with a warning.
func buildSources(cfg *config.Config) ([]source.LogSource, error) {
	var sources []source.LogSource
	for _, sc := range cfg.Sources {
		switch sc.Type {
		case config.SourceTypeDocker:
			s, err := source.NewDockerSource(sc.Name, sc.ComposeFile)
			if err != nil {
				return nil, fmt.Errorf("create source %q: %w", sc.Name, err)
			}
			sources = append(sources, s)
		case config.SourceTypeFile:
			sources = append(sources, source.NewFileSource(sc.Name, sc.Path))
		case config.SourceTypeSystemd:
			s, err := newSystemdSource(sc.Name, sc.Unit)
			if err != nil {
				return nil, fmt.Errorf("create source %q: %w", sc.Name, err)
			}
			if s != nil {
				sources = append(sources, s)
			}
		}
	}
	return sources, nil
}

// exitFunc is overridable in tests to avoid calling os.Exit.
var exitFunc = os.Exit

// Execute is the entry point called from main.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		exitFunc(1)
	}
}
