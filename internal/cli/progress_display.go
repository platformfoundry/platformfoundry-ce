package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/platformfoundry/platformfoundry-ce/internal/engine"
)

// ProgressDisplay shows real-time engine progress
type ProgressDisplay struct {
	engines   map[string]*engineProgress
	mu        sync.RWMutex
	started   bool
	isVerbose bool
}

type engineProgress struct {
	name     string
	state    engine.EngineState
	progress int
	message  string
}

func NewProgressDisplay() *ProgressDisplay {
	return &ProgressDisplay{
		engines:   make(map[string]*engineProgress),
		isVerbose: os.Getenv("PF_VERBOSE") == "true",
	}
}

// Start initializes the display
func (p *ProgressDisplay) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started = true
	if !p.isVerbose {
		fmt.Print("\033[?25l") // Hide cursor
	}
}

// Stop cleans up the display
func (p *ProgressDisplay) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started = false
	if !p.isVerbose {
		fmt.Print("\033[?25h") // Show cursor
		fmt.Println()         // New line after progress display
	}
}

func (p *ProgressDisplay) OnEvent(event engine.EngineEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	if _, ok := p.engines[event.EngineID]; !ok {
		p.engines[event.EngineID] = &engineProgress{name: event.Component}
	}

	ep := p.engines[event.EngineID]

	switch event.Type {
	case engine.EventTypeProgress:
		ep.progress = event.Progress
		if event.Message != "" {
			ep.message = event.Message
		}
	case engine.EventTypeStateChange:
		ep.state = engine.EngineState(event.Message)
	case engine.EventTypeLog:
		ep.message = event.Message
	case engine.EventTypeError:
		ep.state = engine.EngineStateFailed
		if event.Error != nil {
			ep.message = event.Error.Error()
		}
	}

	p.render()
}

func (p *ProgressDisplay) render() {
	// Clear and redraw
	fmt.Print("\033[H\033[2J") // Clear screen

	fmt.Println("Platform Provisioning Progress")
	fmt.Println(strings.Repeat("═", 60))

	for _, ep := range p.engines {
		bar := p.progressBar(ep.progress, 30)
		status := p.stateIcon(ep.state)
		fmt.Printf("%s %-15s [%s] %3d%% %s\n",
			status, ep.name, bar, ep.progress, ep.message)
	}
}

func (p *ProgressDisplay) progressBar(progress, width int) string {
	filled := (progress * width) / 100
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func (p *ProgressDisplay) stateIcon(state engine.EngineState) string {
	switch state {
	case engine.EngineStateCompleted:
		return "✓"
	case engine.EngineStateFailed:
		return "✗"
	case engine.EngineStateRunning:
		return "⟳"
	case engine.EngineStateWaiting:
		return "◌"
	default:
		return "○"
	}
}

func (p *ProgressDisplay) Subscribe(coordinator *engine.Coordinator) {
	// This is a placeholder for where the subscription would happen.
	// In a real implementation, you would pass the coordinator to the NewProgressDisplay
	// and have it subscribe to the coordinator's event bus.
}
