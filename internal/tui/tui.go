package tui

import (
	"context"
	"encoding/json"
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
	// State tracking for log tailing
	lastTimestamp time.Time
	lastTimestampMutex sync.RWMutex
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
		ticker := time.NewTicker(3 * time.Second)  // Use same interval as non-TUI mode
		defer ticker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				return
			case <-ticker.C:
				t.filterMutex.RLock()
				currentTags := t.currentTags
				t.filterMutex.RUnlock()

				// Update config with current filter if needed
				if currentTags != t.config.GetTags() {
					t.config.Tags = currentTags
				}

				// Use the same logic as TailLogs but adapted for TUI
				t.lastTimestampMutex.RLock()
				from := t.lastTimestamp
				t.lastTimestampMutex.RUnlock()

				if from.IsZero() {
					from = time.Now().Add(-1 * time.Minute) // Start with 1 minute window
				} else {
					// Add 1 nanosecond to avoid duplicate logs
					from = from.Add(1 * time.Nanosecond)
				}
				to := time.Now()

				logs, latest, err := t.fetchLogsForTUI(from, to)
				if err != nil {
					t.app.QueueUpdateDraw(func() {
						fmt.Fprintf(t.logView, "[red]Error fetching logs: %v[white]\n", err)
					})
					continue
				}

				// Update lastTimestamp to avoid duplicate logs
				if !latest.IsZero() {
					t.lastTimestampMutex.Lock()
					if t.lastTimestamp.IsZero() || latest.After(t.lastTimestamp) {
						t.lastTimestamp = latest
					}
					t.lastTimestampMutex.Unlock()
				} else if len(logs) == 0 {
					// If no logs returned, advance time slightly to avoid infinite loop
					t.lastTimestampMutex.Lock()
					if t.lastTimestamp.IsZero() {
						t.lastTimestamp = time.Now().Add(-30 * time.Second)
					} else {
						// Move forward by a small amount when no new logs
						t.lastTimestamp = time.Now().Add(-10 * time.Second)
					}
					t.lastTimestampMutex.Unlock()
				}

				// Display logs
				t.displayLogs(logs)
			}
		}
	}()
}

func (t *TUI) fetchLogsForTUI(from, to time.Time) ([]map[string]interface{}, time.Time, error) {
	ctx := context.Background()
	
	// Use the same fetchLogsV2 method as the main TailLogs
	logs, latest, err := t.client.FetchLogsV2(ctx, from, to)
	if err != nil {
		return nil, time.Time{}, err
	}

	// Convert LogEntry to map for TUI display
	var result []map[string]interface{}
	for _, log := range logs {
		logMap := map[string]interface{}{
			"id":         log.GetID(),
			"timestamp":  time.Unix(log.GetTimestamp(), 0).Format("15:04:05"),
			"message":    log.GetMessage(),
			"service":    log.GetService(),
			"level":      log.GetStatus(),
			"tags":       log.GetTags(),
			"attributes": log.GetAttributes(),
		}
		result = append(result, logMap)
	}
	
	return result, latest, nil
}

func (t *TUI) displayLogs(logs []map[string]interface{}) {
	if len(logs) == 0 {
		return
	}

	t.app.QueueUpdateDraw(func() {
		for _, log := range logs {
			// Convert log to JSON format
			jsonBytes, err := json.MarshalIndent(log, "", "  ")
			if err != nil {
				fmt.Fprintf(t.logView, "[red]Error formatting log as JSON: %v[white]\n", err)
				continue
			}
			
			// Add color highlighting to the JSON
			jsonStr := string(jsonBytes)
			
			// Color the JSON output for better readability
			jsonStr = strings.ReplaceAll(jsonStr, `"timestamp":`, `[cyan]"timestamp":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"level":`, `[green]"level":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"service":`, `[yellow]"service":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"message":`, `[white]"message":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"tags":`, `[lightgreen]"tags":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"attributes":`, `[lightblue]"attributes":[white]`)
			jsonStr = strings.ReplaceAll(jsonStr, `"id":`, `[magenta]"id":[white]`)
			
			// Print the colored JSON
			fmt.Fprint(t.logView, jsonStr)
			fmt.Fprint(t.logView, "\n")
			
			// Add separator line for readability
			fmt.Fprint(t.logView, "[darkgray]"+strings.Repeat("─", 80)+"[white]\n")
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