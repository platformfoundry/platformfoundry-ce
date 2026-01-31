package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Bar represents a progress bar
type Bar struct {
	Total       int64
	Current     int64
	Width       int
	Description string
	StartTime   time.Time
	Writer      io.Writer
	mu          sync.Mutex
	done        bool
}

// NewBar creates a new progress bar
func NewBar(total int64, description string) *Bar {
	return &Bar{
		Total:       total,
		Current:     0,
		Width:       40,
		Description: description,
		StartTime:   time.Now(),
		Writer:      os.Stdout,
	}
}

// Increment increments the progress by n
func (b *Bar) Increment(n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Current += n
	if b.Current > b.Total {
		b.Current = b.Total
	}
}

// Set sets the current progress
func (b *Bar) Set(current int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Current = current
	if b.Current > b.Total {
		b.Current = b.Total
	}
}

// Render renders the progress bar
func (b *Bar) Render() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.done {
		return
	}

	percentage := float64(b.Current) / float64(b.Total) * 100
	filled := int(float64(b.Width) * float64(b.Current) / float64(b.Total))
	empty := b.Width - filled

	bar := strings.Repeat("━", filled) + strings.Repeat("░", empty)

	elapsed := time.Since(b.StartTime)
	var eta time.Duration
	if b.Current > 0 {
		rate := float64(b.Current) / elapsed.Seconds()
		remaining := b.Total - b.Current
		eta = time.Duration(float64(remaining)/rate) * time.Second
	}

	fmt.Fprintf(b.Writer, "\r  %s  %s %.0f%%  [%s / ETA: %s]  ",
		b.Description,
		bar,
		percentage,
		formatDuration(elapsed),
		formatDuration(eta),
	)
}

// Finish marks the progress bar as complete
func (b *Bar) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.Current = b.Total
	b.done = true

	percentage := 100.0
	bar := strings.Repeat("━", b.Width)
	elapsed := time.Since(b.StartTime)

	fmt.Fprintf(b.Writer, "\r  %s  %s %.0f%%  [%s]  \n",
		b.Description,
		bar,
		percentage,
		formatDuration(elapsed),
	)
}

// Spinner represents a spinner for indeterminate progress
type Spinner struct {
	Message   string
	frames    []string
	interval  time.Duration
	stopChan  chan bool
	doneChan  chan bool
	mu        sync.Mutex
	isRunning bool
	Writer    io.Writer
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		Message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		stopChan: make(chan bool),
		doneChan: make(chan bool),
		Writer:   os.Stdout,
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	go func() {
		i := 0
		for {
			select {
			case <-s.stopChan:
				s.doneChan <- true
				return
			default:
				frame := s.frames[i%len(s.frames)]
				fmt.Fprintf(s.Writer, "\r  %s %s", frame, s.Message)
				time.Sleep(s.interval)
				i++
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.stopChan <- true
	<-s.doneChan
	s.isRunning = false
	fmt.Fprintf(s.Writer, "\r")
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.Writer, "\r  ✅ %s\n", message)
}

// Failure stops the spinner with a failure message
func (s *Spinner) Failure(message string) {
	s.Stop()
	fmt.Fprintf(s.Writer, "\r  ❌ %s\n", message)
}

// Warning stops the spinner with a warning message
func (s *Spinner) Warning(message string) {
	s.Stop()
	fmt.Fprintf(s.Writer, "\r  ⚠️  %s\n", message)
}

// MultiBar manages multiple progress bars
type MultiBar struct {
	bars   []*Bar
	Writer io.Writer
	mu     sync.Mutex
}

// NewMultiBar creates a new multi-bar manager
func NewMultiBar() *MultiBar {
	return &MultiBar{
		bars:   make([]*Bar, 0),
		Writer: os.Stdout,
	}
}

// AddBar adds a new progress bar
func (m *MultiBar) AddBar(total int64, description string) *Bar {
	m.mu.Lock()
	defer m.mu.Unlock()

	bar := NewBar(total, description)
	bar.Writer = m.Writer
	m.bars = append(m.bars, bar)
	return bar
}

// Render renders all progress bars
func (m *MultiBar) Render() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear previous output
	for i := 0; i < len(m.bars)+2; i++ {
		fmt.Fprintf(m.Writer, "\033[A\033[K")
	}

	// Render all bars
	for _, bar := range m.bars {
		bar.Render()
		fmt.Fprintln(m.Writer)
	}

	// Calculate overall progress
	var totalCurrent, totalMax int64
	for _, bar := range m.bars {
		totalCurrent += bar.Current
		totalMax += bar.Total
	}

	overallPercentage := float64(totalCurrent) / float64(totalMax) * 100
	fmt.Fprintf(m.Writer, "\n  Overall Progress: %.0f%%\n", overallPercentage)
}

// Phase represents a phase in a multi-phase operation
type Phase struct {
	Name        string
	Description string
	Status      PhaseStatus
	StartTime   time.Time
	EndTime     time.Time
	Progress    int // 0-100
	Message     string
}

// PhaseStatus represents the status of a phase
type PhaseStatus string

const (
	PhasePending    PhaseStatus = "pending"
	PhaseRunning    PhaseStatus = "running"
	PhaseCompleted  PhaseStatus = "completed"
	PhaseFailed     PhaseStatus = "failed"
	PhaseSkipped    PhaseStatus = "skipped"
)

// PhaseTracker tracks progress through multiple phases
type PhaseTracker struct {
	Title  string
	Phases []*Phase
	Writer io.Writer
	mu     sync.Mutex
}

// NewPhaseTracker creates a new phase tracker
func NewPhaseTracker(title string) *PhaseTracker {
	return &PhaseTracker{
		Title:  title,
		Phases: make([]*Phase, 0),
		Writer: os.Stdout,
	}
}

// AddPhase adds a new phase
func (pt *PhaseTracker) AddPhase(name string, description string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.Phases = append(pt.Phases, &Phase{
		Name:        name,
		Description: description,
		Status:      PhasePending,
		Progress:    0,
	})
}

// StartPhase starts a phase
func (pt *PhaseTracker) StartPhase(name string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, phase := range pt.Phases {
		if phase.Name == name {
			phase.Status = PhaseRunning
			phase.StartTime = time.Now()
			break
		}
	}
}

// UpdatePhase updates phase progress and message
func (pt *PhaseTracker) UpdatePhase(name string, progress int, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, phase := range pt.Phases {
		if phase.Name == name {
			phase.Progress = progress
			phase.Message = message
			break
		}
	}
}

// CompletePhase marks a phase as completed
func (pt *PhaseTracker) CompletePhase(name string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, phase := range pt.Phases {
		if phase.Name == name {
			phase.Status = PhaseCompleted
			phase.Progress = 100
			phase.EndTime = time.Now()
			break
		}
	}
}

// FailPhase marks a phase as failed
func (pt *PhaseTracker) FailPhase(name string, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, phase := range pt.Phases {
		if phase.Name == name {
			phase.Status = PhaseFailed
			phase.Message = message
			phase.EndTime = time.Now()
			break
		}
	}
}

// Render renders the phase tracker
func (pt *PhaseTracker) Render() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Title
	fmt.Fprintf(pt.Writer, "\n┌─────────────────────────────────────────────┐\n")
	fmt.Fprintf(pt.Writer, "│  %-43s│\n", pt.Title)
	fmt.Fprintf(pt.Writer, "├─────────────────────────────────────────────┤\n")

	// Phases
	for _, phase := range pt.Phases {
		icon := ""
		statusStr := ""

		switch phase.Status {
		case PhasePending:
			icon = "⏳"
			statusStr = "Pending"
		case PhaseRunning:
			icon = "🔄"
			if phase.Progress > 0 {
				bar := strings.Repeat("━", phase.Progress/5) + strings.Repeat("░", 20-phase.Progress/5)
				statusStr = fmt.Sprintf("[%s %d%%]", bar, phase.Progress)
			} else {
				statusStr = "Running..."
			}
		case PhaseCompleted:
			icon = "✅"
			duration := phase.EndTime.Sub(phase.StartTime)
			statusStr = fmt.Sprintf("(%s)", formatDuration(duration))
		case PhaseFailed:
			icon = "❌"
			statusStr = "Failed"
		case PhaseSkipped:
			icon = "⊘"
			statusStr = "Skipped"
		}

		fmt.Fprintf(pt.Writer, "│  %s %-25s %-13s│\n", icon, phase.Description, statusStr)
		if phase.Message != "" && phase.Status == PhaseRunning {
			fmt.Fprintf(pt.Writer, "│     %-39s│\n", truncate(phase.Message, 39))
		}
	}

	// Overall progress
	completed := 0
	for _, phase := range pt.Phases {
		if phase.Status == PhaseCompleted {
			completed++
		}
	}
	overallPercentage := float64(completed) / float64(len(pt.Phases)) * 100

	fmt.Fprintf(pt.Writer, "│                                             │\n")
	bar := strings.Repeat("━", int(overallPercentage/5)) + strings.Repeat("░", 20-int(overallPercentage/5))
	fmt.Fprintf(pt.Writer, "│  Overall Progress: %s %.0f%%       │\n", bar, overallPercentage)
	fmt.Fprintf(pt.Writer, "└─────────────────────────────────────────────┘\n")
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}

	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
