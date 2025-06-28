package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jedipunkz/datadog-log-tail/internal/config"
	"github.com/jedipunkz/datadog-log-tail/internal/datadog"
	"github.com/rivo/tview"
)

type TUI struct {
	app         *tview.Application
	inputField  *tview.InputField
	logView     *tview.TextView
	client      *datadog.Client
	config      *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	filterMutex sync.RWMutex
	currentTags string
}

func New(cfg *config.Config) (*TUI, error) {
	client, err := datadog.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Datadog client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TUI{
		app:    tview.NewApplication(),
		client: client,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (t *TUI) setupUI() {
	// Create input field for filter
	t.inputField = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(0).
		SetPlaceholder("Enter tags (e.g., service:web,env:prod)")

	// Create text view for logs
	t.logView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)

	t.logView.SetBorder(true).SetTitle("Logs")

	// Handle input changes
	t.inputField.SetChangedFunc(func(text string) {
		t.filterMutex.Lock()
		t.currentTags = text
		t.filterMutex.Unlock()
		
		// Clear existing logs when filter changes
		t.logView.Clear()
		t.logView.SetText("Filter updated. New logs will appear here...\n")
	})

	// Create layout
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.inputField, 3, 0, true).
		AddItem(t.logView, 0, 1, false)

	t.app.SetRoot(flex, true)

	// Set up key bindings
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.cancel()
			t.app.Stop()
			return nil
		case tcell.KeyTab:
			// Switch focus between input and log view
			if t.app.GetFocus() == t.inputField {
				t.app.SetFocus(t.logView)
			} else {
				t.app.SetFocus(t.inputField)
			}
			return nil
		case tcell.KeyEscape:
			t.app.SetFocus(t.inputField)
			return nil
		}
		return event
	})
}

func (t *TUI) startLogTailing() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				return
			case <-ticker.C:
				t.filterMutex.RLock()
				currentTags := t.currentTags
				t.filterMutex.RUnlock()

				// Update config with current filter
				tempConfig := *t.config
				tempConfig.Tags = currentTags

				// Get logs from Datadog
				logs, err := t.client.GetLogs(&tempConfig)
				if err != nil {
					t.app.QueueUpdateDraw(func() {
						fmt.Fprintf(t.logView, "[red]Error fetching logs: %v[white]\n", err)
					})
					continue
				}

				// Display logs
				t.displayLogs(logs)
			}
		}
	}()
}

func (t *TUI) displayLogs(logs []map[string]interface{}) {
	if len(logs) == 0 {
		return
	}

	t.app.QueueUpdateDraw(func() {
		for _, log := range logs {
			timestamp := ""
			if ts, ok := log["timestamp"]; ok {
				timestamp = fmt.Sprintf("[blue]%v[white] ", ts)
			}

			level := ""
			if lvl, ok := log["level"]; ok {
				color := getLogLevelColor(fmt.Sprintf("%v", lvl))
				level = fmt.Sprintf("[%s]%v[white] ", color, lvl)
			}

			service := ""
			if svc, ok := log["service"]; ok {
				service = fmt.Sprintf("[yellow]%v[white] ", svc)
			}

			message := ""
			if msg, ok := log["message"]; ok {
				message = fmt.Sprintf("%v", msg)
			}

			logLine := fmt.Sprintf("%s%s%s%s\n", timestamp, level, service, message)
			fmt.Fprint(t.logView, logLine)
		}
		
		// Auto-scroll to bottom
		t.logView.ScrollToEnd()
	})
}

func getLogLevelColor(level string) string {
	switch strings.ToLower(level) {
	case "error":
		return "red"
	case "warn", "warning":
		return "orange"
	case "info":
		return "green"
	case "debug":
		return "gray"
	default:
		return "white"
	}
}

func (t *TUI) Run() error {
	t.setupUI()
	t.startLogTailing()

	// Set initial focus to input field
	t.app.SetFocus(t.inputField)

	// Add instructions
	t.logView.SetText("TUI Mode - Real-time Datadog Log Tail\n" +
		"Instructions:\n" +
		"- Type filter tags in the input field above\n" +
		"- Press Tab to switch between input and log view\n" +
		"- Press Ctrl+C to exit\n" +
		"- Use arrow keys to scroll in log view\n\n" +
		"Waiting for logs...\n")

	return t.app.Run()
}

func (t *TUI) Stop() {
	t.cancel()
	t.app.Stop()
}