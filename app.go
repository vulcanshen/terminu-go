package terminu

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Rect is a panel's place in the panel area, in cells.
type Rect struct{ X, Y, W, H int }

// Screen is a set of panels shown together (Principle "Screen"). A screen's
// panels are numbered from [1] by their order (Rules K1).
type Screen struct {
	// Name is the screen's chip and its row in the global operation popup
	// ("Panels"); Key is its global hotkey ("P"). Desc is that row's
	// description.
	Name, Key, Desc string
	Panels          []*Panel
	// Layout places the panels in the panel area, one Rect each; nil puts
	// them side by side in equal widths. How a screen is laid out is the
	// app's (Rules L1).
	Layout func(w, h int) []Rect
	// NarrowBelow is the width under which only the focused panel is drawn,
	// across the whole width (layout/screen, narrow); at most 80.
	NarrowBelow int

	focus int
	held  *Panel // the panel with the focus, kept when panels come and go
}

// Focused is the panel with the focus.
func (s *Screen) Focused() *Panel {
	if len(s.Panels) == 0 {
		return nil
	}
	return s.Panels[s.at()]
}

// at is the place of the panel with the focus. The focus stays on the same
// panel when the app adds or takes away panels; when that panel itself goes,
// the one now in its place has it.
func (s *Screen) at() int {
	if i := slices.Index(s.Panels, s.held); i >= 0 {
		s.focus = i
		return i
	}
	s.focus = min(max(s.focus, 0), max(len(s.Panels)-1, 0))
	if len(s.Panels) > 0 {
		s.held = s.Panels[s.focus]
	}
	return s.focus
}

// App is the whole terminal UI: screens of panels, the popups over them,
// the footer. It is a Bubble Tea model.
type App struct {
	// Name is the app's name, for its environment variables (Rules E2).
	Name string
	// Quit is the quit flow (Rules K9): nil leaves at once, an Ask confirms
	// first. A second Ctrl-C during the flow always leaves.
	Quit *Ask
	// QuitDesc is the Quit row's description in the global operation popup.
	QuitDesc string
	// QuitFlow, when set, is the quit flow itself, in place of Quit (Rules
	// K9: the app's): q and Ctrl-C run it, a menu of the app's own to choose
	// how to leave, say. While the popup it opens is open, a second Ctrl-C
	// leaves at once, as in any quit flow.
	QuitFlow func() tea.Cmd
	// Global are the global operations besides switching screens and
	// quitting (Principle P3); their hotkeys work on panels too (Rules M4).
	Global []Action
	// Status draws the statusbar in w cells (layout/screen); nil for none.
	Status func(w int) string
	// Canvas, when set, is a screen the app draws itself, whole: a full-screen
	// lock screen. The lib draws it instead of its screens, chips and footer,
	// h lines of w cells, and puts its popups and toasts over it, so confirm,
	// toast and every kind of input work there as anywhere (Rules E8).
	Canvas func(w, h int) []string
	// CanvasKey gets every key on the Canvas while no popup is open and
	// reports whether it took it. Of what it leaves only Ctrl-C does
	// something, the quit flow (Rules K9: App.Quit decides); no Space menu,
	// no ?, no q: the screen is the app's.
	CanvasKey func(key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// HideSecretsInPreview is the user's setting hide_secrets_in_preview,
	// read from the app's settings file (tdp E3): a file preview shows only
	// the first line of a private key, and nothing of a file only its owner
	// may read (components/input/file-picker).
	HideSecretsInPreview bool
	// InputTabWidth is the user's setting input_tab_width: how many spaces
	// Tab puts in a textarea, and the spaces a tab in any text is drawn as;
	// 4 when unset (components/input/textarea, Rules E3).
	InputTabWidth int
	// ToastLines caps how many lines a toast has, the last one kept ending in
	// "…"; 0 for no cap (components/dialog/toast: the app's choice).
	ToastLines int
	// Spinning reports that a loading icon of the app's own is on the screen
	// (LoadingFrame in a row), so the lib keeps drawing it turning.
	Spinning func() bool
	// Progress is a long job's progress, 0 to 1, for the bar under the
	// screen chips; ok false when there is none.
	Progress func() (frac float64, ok bool)
	// ShiftTab makes Shift-Tab go to the previous panel (Rules K2: the app's
	// choice).
	ShiftTab bool
	// OnMsg gets the messages the lib does not handle itself.
	OnMsg func(tea.Msg) tea.Cmd
	// OnResize runs when the terminal changes size, after the lib has laid
	// the screen out again: the place to change what depends on the width
	// (which columns a list shows; Rules L1).
	OnResize func(w, h int) tea.Cmd
	// ForceNarrow draws every screen narrow (for showing the narrow layout).
	ForceNarrow bool
	// Start runs when the program starts.
	Start func() tea.Cmd
	// Splash is the family's easter egg, opened with V on a panel (Rules S).
	// Every app has one: Run refuses an app without it.
	Splash Splash
	// MinWidth and MinHeight are the app's least size (Rules L1): 80 × 40 when
	// unset, the most they may be. On a smaller terminal only the app's name
	// and "needs a terminal of at least W×H" are drawn, in the middle; every
	// key is ignored, the PTY's too, but Ctrl-C pressed twice, which leaves.
	// What the app was showing comes back as the terminal grows.
	MinWidth, MinHeight int
	// CoreKeys are core keys of the app's own (Rules K1): each means the same
	// on every panel and popup, but on a typing row, in a PTY and in a mode;
	// no hotkey may take one.
	CoreKeys []CoreKey

	screens  []*Screen
	cur      int
	stack    []*layer
	toast    *layer
	toastSeq int
	w, h     int
	pending  string // "g" waiting for its second key
	quitting bool
	// quitConfirm is the quit flow's popup while quitting: its confirm, or
	// what the app's QuitFlow opened.
	quitConfirm popupView
	anim        bool
	spin        bool

	splashOn    bool
	splashFrame int
	splashID    int
	shown       Splash    // the splash on the screen: the app's, or one it plays
	splashStart time.Time // when it opened

	tinyCtrlC bool // Ctrl-C pressed once on a terminal under the least size
}

// minSize is the least size: the app's, 80 × 40 at most (Rules L1).
func (a *App) minSize() (w, h int) {
	w, h = 80, 40
	if a.MinWidth > 0 {
		w = min(a.MinWidth, 80)
	}
	if a.MinHeight > 0 {
		h = min(a.MinHeight, 40)
	}
	return w, h
}

// tooSmall reports a terminal under the app's least size.
func (a *App) tooSmall() bool {
	mw, mh := a.minSize()
	return a.w < mw || a.h < mh
}

// tinyScreen is all that is drawn under the least size: the app's name in
// Blue bold and, under it, what size it needs in Overlay0, in the middle;
// with one row, or no room for the second, the name alone (components/layout/
// screen, smaller than the least size).
func (a *App) tinyScreen(w, h int) []string {
	mw, mh := a.minSize()
	need := "needs a terminal of at least " + itoa(mw) + "×" + itoa(mh)
	// As much of the name as fits, nothing put in its place.
	rows := []string{Fg(Blue).Bolded().Render(clip(a.Name, w))}
	if h >= 2 && Width(need) <= w {
		rows = append(rows, Fg(Overlay0).Render(need))
	}
	out := blankLines(w, h)
	top := (h - len(rows)) / 2
	for i, r := range rows {
		out[top+i] = Center(r, w)
	}
	return out
}

// New makes an app called name.
func New(name string) *App { return &App{Name: name} }

// AddScreen adds a screen. It panics when a panel registers a key that is
// not the app's to take (Rules K1, K12): that is a mistake in the program.
func (a *App) AddScreen(s *Screen) *Screen {
	for i, p := range s.Panels {
		p.app = a
		for _, act := range p.Actions {
			if err := checkPanelKey(act.Key); err != nil {
				panic(fmt.Sprintf("terminu: panel [%d] %s, action %q: %v", i+1, p.Label, act.Label, err))
			}
			if err := checkToDetails(s, p, act); err != nil {
				panic(fmt.Sprintf("terminu: panel [%d] %s, action %q: %v", i+1, p.Label, act.Label, err))
			}
		}
	}
	a.screens = append(a.screens, s)
	return s
}

// checkToDetails refuses an action that sends the focus to the details
// anywhere but the one place Rules K3 allows: Enter, on a panel whose Details
// is another panel of the same screen.
func checkToDetails(s *Screen, p *Panel, act Action) error {
	switch {
	case act.ToDetails == nil:
		return nil
	case act.Key != "Enter":
		return fmt.Errorf("only Enter may send the focus to the details (Rules K3)")
	case p.Details == nil || p.Details == p || !slices.Contains(s.Panels, p.Details):
		return fmt.Errorf("its panel's Details is not another panel of the screen (Rules K3)")
	}
	return nil
}

// checkPanelKey refuses a hotkey that a core key or a movement letter owns
// on a panel.
// checkPopupKey is checkPanelKey for a popup's hotkey: the digits are free
// there (Rules K1).
func checkPopupKey(key string) error {
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		return nil
	}
	return checkPanelKey(key)
}

func checkPanelKey(key string) error {
	switch {
	case key == "" || key == "Enter":
		return nil
	case slices.Contains([]string{"Tab", "Esc", "Space", "?", "q", "Ctrl-C", "gg"}, key):
		return fmt.Errorf("%s is a core key (Rules K1)", key)
	case len(key) == 1 && key[0] >= '1' && key[0] <= '9':
		return fmt.Errorf("the digits jump to panels (Rules K1)")
	case slices.Contains([]string{"j", "k", "u", "d", "g", "G", "h", "l"}, key):
		return fmt.Errorf("%s is kept for moving (Rules K12)", key)
	}
	return nil
}

// Screen is the current screen.
func (a *App) Screen() *Screen {
	if len(a.screens) == 0 {
		return nil
	}
	return a.screens[a.cur]
}

// Size is the terminal's size.
func (a *App) Size() (w, h int) { return a.w, a.h }

// Run asks the terminal how wide an icon is and what its colours are, then
// runs the app in the alternate screen.
func (a *App) Run() error {
	if err := a.checkSplash(); err != nil {
		return err
	}
	if err := a.checkKeys(); err != nil {
		return err
	}
	Probe(a.Name)
	tabCells = a.inputTabWidth()
	_, err := tea.NewProgram(a, tea.WithAltScreen(), tea.WithOutput(screenOutput{os.Stdout})).Run()
	return err
}

// Init runs Start.
func (a *App) Init() tea.Cmd { return run(a.Start) }

// Update handles a message.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		a.relayout()
		if a.OnResize != nil {
			cmd = a.OnResize(a.w, a.h)
		}
	case animMsg:
		cmd = a.stepAnimation()
	case spinMsg:
		a.spin = false
	case splashMsg:
		cmd = a.stepSplash(msg)
	case toastDone:
		if a.toast != nil && a.toast.view.(*toastView).id == msg.id && !a.toast.closing {
			a.toast.closing = true
			cmd = a.animate()
		}
	case tea.KeyMsg:
		cmd = a.handleKey(msg)
	case formRan:
		cmd = msg.done(a)
	case pauseTick:
		cmd = msg.li.countDown()
	case termOutput, termExited:
		cmd, _ = a.terminalMsg(msg)
	default:
		if a.OnMsg != nil {
			cmd = a.OnMsg(msg)
		}
	}
	cmd = tea.Batch(cmd, a.moved())
	if !a.spin && a.loading() {
		a.spin = true
		cmd = tea.Batch(cmd, spinTick())
	}
	a.growLoading()
	return a, cmd
}

// moved runs the OnMove of the screen's lists and value panels whose cursor
// came to another item: after every message and before the frame is drawn
// (layout/panel, details).
func (a *App) moved() tea.Cmd {
	s := a.Screen()
	if s == nil {
		return nil
	}
	var cmds []tea.Cmd
	for _, p := range s.Panels {
		if b, ok := p.Body.(interface{ moved() tea.Cmd }); ok {
			cmds = append(cmds, b.moved())
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) inputTabWidth() int {
	if a.InputTabWidth > 0 {
		return a.InputTabWidth
	}
	return 4
}

// growLoading measures again a popup that is loading, and once more when it
// stops: only then may its height change while it is open (Rules F7).
func (a *App) growLoading() {
	for _, l := range a.stack {
		v, ok := l.view.(interface{ isLoading() bool })
		if !ok {
			continue
		}
		now := v.isLoading()
		if now || l.loading {
			l.w, l.h = l.view.measure(a.w, a.h)
			l.place(a.w, a.h)
		}
		l.loading = now
	}
}

// loading reports whether a loading icon is on screen, so it keeps turning.
func (a *App) loading() bool {
	if a.Spinning != nil && a.Spinning() {
		return true
	}
	if s := a.Screen(); s != nil {
		for _, p := range s.Panels {
			if p.Loading {
				return true
			}
		}
	}
	for _, l := range a.stack {
		if v, ok := l.view.(interface{ isLoading() bool }); ok && v.isLoading() {
			return true
		}
	}
	return false
}

// --- popups ------------------------------------------------------------------

// open puts v on top of the stack and starts its animation.
func (a *App) open(v popupView) tea.Cmd {
	l := &layer{view: v, frame: 1}
	l.w, l.h = v.measure(a.w, a.h)
	l.place(a.w, a.h)
	a.stack = append(a.stack, l)
	return a.animate()
}

func (a *App) animate() tea.Cmd {
	if a.anim {
		return nil
	}
	a.anim = true
	return animTick()
}

// stepAnimation moves every animating popup on by one frame.
func (a *App) stepAnimation() tea.Cmd {
	busy := false
	var cmds []tea.Cmd
	step := func(l *layer) bool { // false when the layer is gone
		switch {
		case l.closing:
			l.frame--
			if l.frame <= 0 {
				if l.below != nil {
					cmds = append(cmds, l.below())
				}
				return false
			}
			busy = true
		case l.frame < openFrames:
			l.frame++
			busy = busy || l.frame < openFrames
		}
		return true
	}
	kept := a.stack[:0]
	for _, l := range a.stack {
		if step(l) {
			kept = append(kept, l)
		}
	}
	a.stack = kept
	if a.toast != nil && !step(a.toast) {
		a.toast = nil
	}
	a.anim = busy
	if busy {
		cmds = append(cmds, animTick())
	}
	return tea.Batch(cmds...)
}

// top is the top popup that is not closing.
func (a *App) top() *layer {
	for i := len(a.stack) - 1; i >= 0; i-- {
		if !a.stack[i].closing {
			return a.stack[i]
		}
	}
	return nil
}

// closeTop starts closing the top popup (Rules F3, K4).
func (a *App) closeTop() tea.Cmd {
	l := a.top()
	if l == nil {
		return nil
	}
	return a.closeLayer(l)
}

// layerOf is l's layer: the open popups under it, and itself (components/color).
func (a *App) layerOf(l *layer) int {
	n := 1
	for _, below := range a.stack {
		if below == l {
			break
		}
		if !below.closing {
			n++
		}
	}
	return n
}

// closeOnly starts closing v's popup alone: a popup opened over it stays (a
// form's submit that opens the next popup).
func (a *App) closeOnly(v popupView) tea.Cmd {
	for _, l := range a.stack {
		if l.view == v && !l.closing {
			return a.closeLayer(l)
		}
	}
	return nil
}

func (a *App) closeLayer(l *layer) tea.Cmd {
	if a.isQuitConfirm(l) {
		a.quitting = false
	}
	// It closes in the colour of the layer it was (Rules F2).
	if !l.shade.Set {
		l.shade = LayerColor(a.layerOf(l))
	}
	l.closing = true
	if l.frame > openFrames {
		l.frame = openFrames
	}
	return tea.Batch(gone(l), a.animate())
}

// gone tells a popup that wants to know it is closing: a form stops what
// still runs for it.
func gone(l *layer) tea.Cmd {
	if c, ok := l.view.(interface{ closed() }); ok {
		c.closed()
	}
	// A popup that closes with no answer was cancelled (OnCancel).
	if c, ok := l.view.(interface{ cancel() tea.Cmd }); ok && !l.answered {
		l.answered = true
		return c.cancel()
	}
	return nil
}

// commitFailed is a popup's write-back failing: an error popup over it,
// the popup and its value staying (components/layout/popup, the action
// failing).
func (a *App) commitFailed(title, subject string, err error) tea.Cmd {
	return a.open(errorPopup(title+" failed", subject, err.Error()))
}

// runCommit runs a popup's Commit, when it has one.
func runCommit[T any](f func(T) error, v T) error {
	if f == nil {
		return nil
	}
	return f(v)
}

// runDone runs a popup's Done, when it has one.
func runDone[T any](f func(T) tea.Cmd, v T) tea.Cmd {
	if f == nil {
		return nil
	}
	return f(v)
}

// answer closes the top popup on its answer, so it is not cancelled.
func (a *App) answer() tea.Cmd {
	if l := a.top(); l != nil {
		l.answered = true
	}
	return a.closeTop()
}

// closeView closes v's popup and every one above it, all together, as
// closeAll does with the whole stack.
func (a *App) closeView(v popupView) tea.Cmd {
	i := slices.IndexFunc(a.stack, func(l *layer) bool { return l.view == v && !l.closing })
	if i < 0 {
		return nil
	}
	return a.closeLayers(a.stack[i:])
}

// closeAll closes every popup (Rules T1, when what was done ends the whole
// stack): each plays its own closing at the same time, at its own size and
// in its own layer's colour, about 288ms in all (Rules F2).
func (a *App) closeAll() tea.Cmd {
	a.quitting = false
	return a.closeLayers(a.stack)
}

// closeUnder closes every popup under top as top opens: the sources it
// clears, all together (Rules T1, F2).
func (a *App) closeUnder(top *layer) tea.Cmd {
	i := slices.Index(a.stack, top)
	if i < 0 {
		return nil
	}
	return a.closeLayers(a.stack[:i])
}

// closeLayers starts closing the layers of ls that are not closing yet.
func (a *App) closeLayers(ls []*layer) tea.Cmd {
	var open []*layer
	for _, l := range ls {
		if !l.closing {
			open = append(open, l)
		}
	}
	// Each keeps the colour of the layer it was, the ones under it closing
	// too (Rules F2).
	for _, l := range open {
		l.shade = LayerColor(a.layerOf(l))
	}
	var cmds []tea.Cmd
	for _, l := range open {
		cmds = append(cmds, a.closeLayer(l))
	}
	return tea.Batch(cmds...)
}

func (a *App) relayout() {
	for _, l := range a.stack {
		l.w, l.h = l.view.measure(a.w, a.h)
		l.place(a.w, a.h)
	}
	if a.toast != nil {
		a.placeToast()
	}
}

// Toast shows a short message from the bottom (components/dialog/toast); a
// second one replaces the first and starts its time again.
func (a *App) Toast(kind ToastKind, text string) tea.Cmd {
	a.toastSeq++
	v := &toastView{kind: kind, text: text, id: a.toastSeq, most: a.ToastLines}
	if a.toast != nil && !a.toast.closing {
		a.toast.view = v
	} else {
		a.toast = &layer{view: v, frame: 1, toast: true}
	}
	a.placeToast()
	id := a.toastSeq
	return tea.Batch(a.animate(), tea.Tick(kind.duration(), func(time.Time) tea.Msg { return toastDone{id} }))
}

// placeToast puts the toast's bottom border on the third row from the bottom.
func (a *App) placeToast() {
	l := a.toast
	l.w, l.h = l.view.measure(a.w, a.h)
	l.x, l.y = (a.w-l.w)/2, a.h-2-l.h
}

// Confirm asks ask; accept runs when the user answers Enter. The popup under
// it stays (Rules T1, a short confirm).
func (a *App) Confirm(ask Ask, accept func() tea.Cmd) tea.Cmd {
	c := newConfirm(ask, nil)
	c.accept = func() tea.Cmd { return tea.Batch(a.closeTop(), run(accept)) }
	return a.open(c)
}

// Error opens an error popup over the top popup (components/dialog/note): an
// action failed.
func (a *App) Error(title, message string) tea.Cmd { return a.ErrorAbout(title, "", message) }

// ErrorAbout is Error naming what failed: subject goes after ": " in the
// title, kept as it is, RENAME FAILED: notes.md (components/dialog/note).
func (a *App) ErrorAbout(title, subject, message string) tea.Cmd {
	return a.open(errorPopup(title, subject, message))
}

// Menu opens a menu of sections (Rules M8); running a row closes the menu
// and everything under it.
func (a *App) Menu(title string, sections ...MenuSection) tea.Cmd {
	return a.MenuAbout(title, "", sections...)
}

// MenuAbout is Menu for one thing: subject goes after ": " in the title,
// kept as it is, OPEN WITH: notes.md (components, upper case).
func (a *App) MenuAbout(title, subject string, sections ...MenuSection) tea.Cmd {
	for _, s := range sections {
		for _, act := range s.Actions {
			if err := checkPopupKey(act.Key); err != nil {
				panic(fmt.Sprintf("terminu: menu %s, action %q: %v", title, act.Label, err))
			}
		}
	}
	m := newMenu(title, sections, nil)
	m.subject = subject
	m.run = func(a *App, act Action) tea.Cmd { return a.runAction(act, true) }
	return a.open(m)
}

func run(f func() tea.Cmd) tea.Cmd {
	if f == nil {
		return nil
	}
	return f()
}

func (a *App) acceptConfirm(c *confirmView) tea.Cmd {
	if l := a.top(); l != nil && l.view == c {
		l.answered = true
	}
	return run(c.accept)
}

// runAction runs an action (Rules F6: asking first when it asks). From a menu,
// it closes the menu and what is under it.
func (a *App) runAction(act Action, fromMenu bool) tea.Cmd {
	if act.disabled() || act.hidden() {
		return nil
	}
	do := func() tea.Cmd {
		into := act.ToDetails != nil && act.ToDetails()
		var cmd tea.Cmd
		if act.Do != nil {
			cmd = act.Do()
		}
		if into {
			a.toDetails()
		}
		return cmd
	}
	// then does it where it was asked for: from a menu, what it opens keeps
	// the menu under it unless it clears it; if it opens nothing the menu's
	// work is done (Rules T1).
	then := func() tea.Cmd {
		if !fromMenu || act.Stay || act.Detail {
			return do()
		}
		before := a.top()
		cmd := do()
		switch top := a.top(); {
		case top == nil || top == before:
			return tea.Batch(a.closeAll(), cmd)
		case act.Clear:
			return tea.Batch(a.closeUnder(top), cmd)
		}
		return cmd
	}
	ask := act.Ask
	if act.AskFor != nil {
		ask = act.AskFor()
	}
	if ask != nil {
		c := newConfirm(*ask, nil)
		// The confirm is only a door: accepted, it goes, and what the action
		// does decides about the menu (Rules T1).
		c.accept = func() tea.Cmd { return tea.Batch(a.answer(), then()) }
		return a.open(c)
	}
	return then()
}

// --- quitting ----------------------------------------------------------------

func (a *App) quitAction() Action {
	desc := a.QuitDesc
	if desc == "" {
		desc = "leave " + a.Name
	}
	return Action{Key: "q", Label: "Quit", Desc: desc, Scope: GlobalScope, Do: a.startQuit}
}

// startQuit is the quit flow (Rules K9). What a popup of the app's own would
// lose goes last in the confirm, in Peach; it asks then even when Quit is nil.
func (a *App) startQuit() tea.Cmd {
	if a.QuitFlow != nil {
		if a.quitting {
			return nil
		}
		before := a.top()
		cmd := a.QuitFlow()
		if top := a.top(); top != nil && top != before {
			// The flow is on until what it opened closes.
			a.quitting, a.quitConfirm = true, top.view
		}
		return cmd
	}
	notes := a.quitNotes()
	if a.Quit == nil && len(notes) == 0 {
		return tea.Quit
	}
	if a.quitting {
		return nil
	}
	ask := Ask{Question: "Quit " + a.Name + "?", Verb: "quit"}
	if a.Quit != nil {
		ask = *a.Quit
	}
	ask.Title = cmpOr(ask.Title, "Quit")
	if len(notes) > 0 {
		ask.Details, ask.Warn = slices.Clone(ask.Details), slices.Clone(ask.Warn)
		if len(ask.Details) > 0 {
			ask.Details = append(ask.Details, "")
		}
		for _, n := range notes {
			ask.Warn = append(ask.Warn, len(ask.Details))
			ask.Details = append(ask.Details, n)
		}
	}
	a.quitting = true
	a.quitConfirm = newConfirm(ask, func() tea.Cmd { return tea.Quit })
	return a.open(a.quitConfirm)
}

// quitNotes are the QuitNote of the popups of the app's own that are open,
// from the bottom.
func (a *App) quitNotes() []string {
	var out []string
	for _, l := range a.stack {
		if v, ok := l.view.(*appPopup); ok && !l.closing {
			if q, ok := v.p.Body.(interface{ QuitNote() string }); ok && q.QuitNote() != "" {
				out = append(out, q.QuitNote())
			}
		}
	}
	return out
}

func (a *App) isQuitConfirm(l *layer) bool {
	return a.quitting && l.view == a.quitConfirm
}

func cmpOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// --- the Space menu and the global operation popup -------------------------

func (a *App) panelActions(p *Panel, scope Scope) []Action {
	var out []Action
	own := map[string]bool{}
	for _, act := range p.Actions {
		own[act.Key] = true
		if act.Scope == scope {
			out = append(out, act)
		}
	}
	// The body's own come after; a key the app registered itself is the app's.
	if b, ok := p.Body.(interface{ Actions() []Action }); ok {
		for _, act := range b.Actions() {
			if act.Scope == scope && (act.Key == "" || !own[act.Key]) {
				out = append(out, act)
			}
		}
	}
	return out
}

// openSpaceMenu opens the focused panel's Space menu (Rules M2).
func (a *App) openSpaceMenu() tea.Cmd {
	s := a.Screen()
	p := s.Focused()
	var sections []MenuSection
	if p.Body != nil && p.Body.HasItem() {
		sections = append(sections, MenuSection{Title: "item operation", Actions: a.panelActions(p, ItemScope)})
	}
	sections = append(sections, MenuSection{Title: "panel operation", Actions: a.panelActions(p, PanelScope)})
	global := Action{Label: "Global operation", Desc: "actions for the whole app", Do: a.openGlobal}
	m := newMenu("", sections, []Action{global})
	m.cased = p.title(s.at() + 1)
	m.space = true
	m.name = "Space menu"
	m.run = func(a *App, act Action) tea.Cmd {
		if act.Label == "Global operation" && act.Key == "" {
			return act.Do()
		}
		return a.runAction(act, true)
	}
	return a.open(m)
}

// globalActions are every global operation: switching screens, the app's
// own, quitting (Rules M4).
func (a *App) globalActions() []Action {
	var out []Action
	if len(a.screens) > 1 {
		for i, s := range a.screens {
			i := i
			desc := s.Desc
			if desc == "" {
				desc = "switch to this screen"
			}
			out = append(out, Action{Key: s.Key, Label: s.Name, Desc: desc, Scope: GlobalScope,
				Disabled: func() bool { return a.cur == i },
				Do:       func() tea.Cmd { return a.switchScreen(i) }})
		}
	}
	out = append(out, a.Global...)
	return append(out, a.quitAction())
}

func (a *App) openGlobal() tea.Cmd {
	m := newMenu("Global operation", []MenuSection{{Actions: a.globalActions()}}, nil)
	m.name = "Global operation"
	m.run = func(a *App, act Action) tea.Cmd {
		if act.Label == "Quit" && act.Key == "q" {
			return act.Do()
		}
		return a.runAction(act, true)
	}
	return a.open(m)
}

func (a *App) switchScreen(i int) tea.Cmd {
	if i < 0 || i >= len(a.screens) || i == a.cur {
		return nil
	}
	a.cur = i
	a.focusChanged()
	return nil
}

// Focus moves the focus to p, on whichever screen it is, switching to that
// screen. The app does it where the user expects the focus to move: after
// what they pressed, never on something that happened in the background (a
// load done, a job ended; Rules K3). A panel on no screen is left alone.
func (a *App) Focus(p *Panel) tea.Cmd {
	for i, s := range a.screens {
		if j := slices.Index(s.Panels, p); j >= 0 {
			cmd := a.switchScreen(i)
			return tea.Batch(cmd, a.focusPanel(s, j))
		}
	}
	return nil
}

// ShowScreen switches to s, as its chip's key does (Rules K3: when the user
// expects it).
func (a *App) ShowScreen(s *Screen) tea.Cmd {
	return a.switchScreen(slices.Index(a.screens, s))
}

// --- the key reference -------------------------------------------------------

// panelKeyRef is what ? lists on a panel (Rules K6, components/dialog/note).
func (a *App) panelKeyRef(p *Panel) *noteView {
	s := a.Screen()
	row := func(act Action) KeyRow {
		return KeyRow{Key: act.Key, Desc: plainLabel(act) + descTail(act.Desc), Disabled: act.disabled()}
	}
	var item, panel []KeyRow
	own := map[string]bool{}
	if p.Body != nil && p.Body.HasItem() {
		for _, act := range a.panelActions(p, ItemScope) {
			if act.Key != "" && !act.hidden() {
				item = append(item, row(act))
				own[act.Key] = true
			}
		}
	}
	for _, act := range a.panelActions(p, PanelScope) {
		if act.Key != "" && !act.hidden() {
			panel = append(panel, row(act))
			own[act.Key] = true
		}
	}
	var move []KeyRow
	if p.Body != nil {
		keys := p.Body.MoveKeys()
		if b, ok := p.Body.(interface{ tabMoveKeys() []KeyRow }); ok && len(p.Tabs) > 0 {
			keys = b.tabMoveKeys() // h|l go to the tabs (Rules K12)
		}
		move = append(move, keys...)
	}
	if len(p.Tabs) > 0 {
		move = append(move, KeyRow{Key: "h|l", Desc: "previous tab, next tab"})
	}
	core := []KeyRow{{Key: "Space", Desc: "what you can do here"}, {Key: "?", Desc: "this list"}}
	if len(s.Panels) > 1 {
		tab := KeyRow{Key: "Tab", Desc: "next panel"}
		if a.ShiftTab {
			tab = KeyRow{Key: "Tab|Shift-Tab", Desc: "next panel, previous panel"} // Rules K6
		}
		core = append(core, tab, KeyRow{Key: "1–" + itoa(len(s.Panels)), Desc: "jump to panel N"})
	}
	if l, ok := p.Body.(*List); ok && l.Filtering() {
		core = append(core, KeyRow{Key: "Esc", Desc: "clear the filter"})
	} else if p.OnEsc != nil {
		core = append(core, KeyRow{Key: "Esc", Desc: "one level up"})
	}
	for _, ck := range a.CoreKeys {
		core = append(core, KeyRow{Key: ck.Key, Desc: ck.Desc})
	}
	core = append(core, KeyRow{Key: "q|Ctrl-C", Desc: "quit"})
	var app []KeyRow
	for _, act := range a.globalActions() {
		// On a panel a digit jumps to a panel: a global one works only in the
		// global operation popup (Rules K1), so it is not a key here.
		if act.Key == "" || act.Key == "q" || isDigit(act.Key) || own[act.Key] || act.hidden() {
			continue
		}
		app = append(app, row(act))
	}
	var sections []refSection
	if v, ok := p.Body.(*Values); ok {
		// What each row is, first, when the app says (layout/panel).
		var rows []KeyRow
		for _, f := range v.Fields {
			if f.Desc != "" {
				rows = append(rows, KeyRow{Key: Upper(f.Label), Desc: f.Desc})
			}
		}
		sections = append(sections, refSection{title: "what each row is", rows: rows, names: true})
	}
	sections = append(sections, []refSection{
		{title: "item operation", rows: item},
		{title: "panel operation", rows: panel},
		{title: "move", rows: move},
		{title: "core keys", rows: core},
		{title: "app-wide", rows: app},
	}...)
	// A section about another surface goes last (components/dialog/note).
	for _, o := range p.OtherKeys {
		sections = append(sections, refSection{title: o.Title, rows: o.Rows})
	}
	return keyReference(p.title(s.at()+1), sections)
}

// refTitle is a popup's name in the title of its key reference: upper case,
// as a title is; a viewer or a terminal keeps the name of what it shows as
// it is (components, upper case).
func refTitle(v popupView) string {
	if t, ok := v.(interface{ refTitle() string }); ok {
		return t.refTitle()
	}
	return Upper(v.surface())
}

// modeKeyRef is a mode's key reference (Rules K11).
func (a *App) modeKeyRef(p *Panel) *noteView {
	var rows []KeyRow
	for _, k := range p.mode.Keys {
		rows = append(rows, KeyRow{Key: k.Key, Desc: k.Desc})
	}
	rows = append(rows, KeyRow{Key: "Esc", Desc: cmpOr(p.mode.Exit, "leave") + ", back to before " + p.mode.Name})
	core := []KeyRow{{Key: "?", Desc: "this list"}, {Key: "q|Ctrl-C", Desc: "quit"}}
	return keyReference(Upper(p.mode.Name), []refSection{{title: p.mode.Name, rows: rows}, {title: "core keys", rows: core}})
}

// --- keys --------------------------------------------------------------------

func (a *App) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := KeyName(msg)
	// Under the least size nothing can be seen: no key acts, none goes to a
	// PTY; only Ctrl-C twice leaves at once (Rules L1, K9).
	if a.tooSmall() {
		if key == "Ctrl-C" && a.tinyCtrlC {
			return tea.Quit
		}
		a.tinyCtrlC = key == "Ctrl-C"
		return nil
	}
	a.tinyCtrlC = false
	// Any key only closes the splash, Ctrl-C too (Rules S3).
	if a.splashOn {
		a.splashOn = false
		return nil
	}
	// A popup that is still opening takes no keys (Rules F2).
	if l := a.top(); l != nil && l.frame < openFrames {
		return nil
	}
	// In a PTY every key goes to the program, Esc and Ctrl-C too, but the
	// exit key and the app's own keys (Rules K10).
	if t := a.ptyFocus(); t != nil {
		return t.key(a, key, msg)
	}
	// The second Ctrl-C of a quit flow leaves at once (Rules K9).
	if key == "Ctrl-C" && a.quitting {
		return tea.Quit
	}
	// Esc takes the toast first (Rules K4, F3).
	if key == "Esc" && a.toast != nil && !a.toast.closing {
		a.toast.closing = true
		if a.toast.frame > openFrames {
			a.toast.frame = openFrames
		}
		return a.animate()
	}
	if a.Canvas != nil && a.top() == nil {
		if a.CanvasKey != nil {
			if done, cmd := a.CanvasKey(key, msg); done {
				return cmd
			}
		}
		if key == "Ctrl-C" {
			return a.startQuit()
		}
		return nil
	}
	if key == "Ctrl-C" {
		return a.startQuit()
	}
	if ck := a.coreKey(key); ck != nil && !a.typingOrMode() {
		a.pending = ""
		return run(ck.Do)
	}
	if l := a.top(); l != nil {
		return a.popupKey(l, key, msg)
	}
	return a.panelKey(key, msg)
}

// popupKey is a key on the top popup: its own, then the core keys it leaves.
func (a *App) popupKey(l *layer, key string, msg tea.KeyMsg) tea.Cmd {
	if !l.view.typing() {
		if k, ok := a.prefix(key, func(pair string) bool { return refHas(l.view.keyRef(), pair) }); ok {
			key = k
		} else {
			return nil
		}
	}
	if done, cmd := l.view.key(a, key, msg); done {
		return cmd
	}
	if l.view.typing() {
		return nil
	}
	switch key {
	case "Esc":
		return a.closeTop()
	case "?":
		return a.open(keyReference(refTitle(l.view), l.view.keyRef()))
	case "q":
		return a.startQuit()
	}
	return nil
}

// prefix joins g with the key after it ("gg", "go"); ok is false while the g
// waits. A lone g does nothing, and a key after it that makes no pair known
// here acts as if the g had not been pressed (Rules K12); a core key after
// it is always itself (K1).
func (a *App) prefix(key string, known func(pair string) bool) (string, bool) {
	if a.pending == "g" {
		a.pending = ""
		if pair := "g" + key; !a.isCoreKey(key) && (pair == "gg" || known(pair)) {
			return pair, true
		}
		return key, true
	}
	if key == "g" {
		a.pending = "g"
		return "", false
	}
	return key, true
}

// panelKey is a key with no popup open: the focused panel's.
func (a *App) panelKey(key string, msg tea.KeyMsg) tea.Cmd {
	s := a.Screen()
	if s == nil {
		return nil
	}
	p := s.Focused()
	if p == nil {
		if key == "q" {
			return a.startQuit()
		}
		return nil
	}
	if p.typing() {
		if key == "Esc" {
			if l, ok := p.Body.(*List); ok {
				l.ClearFilter()
			}
			return nil
		}
		_, cmd := p.Body.Key(key, msg)
		return cmd
	}
	if p.mode != nil {
		return a.modeKey(p, key, msg)
	}
	k, ok := a.prefix(key, func(pair string) bool { return a.panelPair(p, pair) })
	if !ok {
		return nil
	}
	key = k
	switch key {
	case "Space":
		return a.openSpaceMenu()
	case "?":
		return a.open(a.panelKeyRef(p))
	case "q":
		return a.startQuit()
	case "Tab":
		return a.focusPanel(s, (s.at()+1)%len(s.Panels))
	case "Shift-Tab":
		if a.ShiftTab {
			return a.focusPanel(s, (s.at()-1+len(s.Panels))%len(s.Panels))
		}
	case "Esc":
		if l, ok := p.Body.(*List); ok && l.Filtering() {
			l.ClearFilter()
			return nil
		}
		if p.OnEsc != nil {
			return p.OnEsc()
		}
		return nil
	case "Enter":
		if act := p.enterAction(); act != nil {
			return a.runAction(*act, false)
		}
	}
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		if n := int(key[0] - '1'); n < len(s.Panels) {
			return a.focusPanel(s, n)
		}
		return nil
	}
	if len(p.Tabs) > 0 && (key == "h" || key == "l" || key == "←" || key == "→") {
		n := len(p.Tabs) + 1
		if key == "h" || key == "←" {
			p.Tab = (p.Tab - 1 + n) % n
		} else {
			p.Tab = (p.Tab + 1) % n
		}
		if p.OnTab != nil {
			return p.OnTab(p.Tab)
		}
		return nil
	}
	// The panel's own hotkeys win over the global ones (Rules M4).
	for _, scope := range []Scope{ItemScope, PanelScope} {
		for _, act := range a.panelActions(p, scope) {
			if act.Key == key && key != "Enter" && !act.hidden() && p.scopeApplies(act) {
				return a.runAction(act, false)
			}
		}
	}
	if key == "V" && a.Splash != nil && !p.takesV() {
		return a.openSplash()
	}
	if p.Body != nil {
		if done, cmd := p.Body.Key(key, msg); done {
			return cmd
		}
	}
	for _, act := range a.globalActions() {
		if act.Key == key && !isDigit(key) {
			if act.Label == "Quit" {
				return a.startQuit()
			}
			return a.runAction(act, false)
		}
	}
	return nil
}

func isDigit(k string) bool { return len(k) == 1 && k[0] >= '0' && k[0] <= '9' }

// panelPair reports whether a g pair is a hotkey on p: its own or a global.
func (a *App) panelPair(p *Panel, pair string) bool {
	acts := append(a.panelActions(p, ItemScope), a.panelActions(p, PanelScope)...)
	for _, act := range append(acts, a.globalActions()...) {
		if act.Key == pair {
			return true
		}
	}
	return false
}

// refHas reports whether a key reference lists key, alone or joined with |.
func refHas(secs []refSection, key string) bool {
	for _, s := range secs {
		for _, r := range s.rows {
			if slices.Contains(strings.Split(r.Key, "|"), key) {
				return true
			}
		}
	}
	return false
}

// modeKey is a key inside a mode (Rules K11).
func (a *App) modeKey(p *Panel, key string, msg tea.KeyMsg) tea.Cmd {
	switch {
	case key == "Space":
		return nil
	case key == "?":
		return a.open(a.modeKeyRef(p))
	case key == "Esc":
		m := p.mode
		p.mode = nil
		return run(m.OnExit)
	case key == "q":
		return a.startQuit()
	case key == "Tab" || isDigit(key) && key != "0":
		if p.mode.Pause {
			return a.Toast(Info, "Press [Esc] to leave "+p.mode.Name+" first")
		}
		s := a.Screen()
		if key == "Tab" {
			return a.focusPanel(s, (s.at()+1)%len(s.Panels))
		}
		if n := int(key[0] - '1'); n < len(s.Panels) {
			return a.focusPanel(s, n)
		}
		return nil
	}
	if key == "V" && a.Splash != nil && !p.takesV() {
		return a.openSplash()
	}
	for _, k := range p.mode.Keys {
		if k.takes(key) && k.Do != nil {
			return k.Do(key)
		}
	}
	return nil
}

// toDetails sends the focus from the focused panel to its Details (Rules K3).
func (a *App) toDetails() {
	s := a.Screen()
	if s == nil || s.Focused() == nil {
		return
	}
	if i := slices.Index(s.Panels, s.Focused().Details); i >= 0 {
		a.focusPanel(s, i)
	}
}

// focusPanel moves the focus to panel i of s.
func (a *App) focusPanel(s *Screen, i int) tea.Cmd {
	if i == s.at() {
		return nil
	}
	s.focus, s.held = i, s.Panels[i]
	a.focusChanged()
	return nil
}

func (a *App) focusChanged() {
	if p := a.Screen().Focused(); p != nil {
		if f, ok := p.Body.(interface{ Focus() }); ok {
			f.Focus()
		}
	}
}

// --- drawing -----------------------------------------------------------------

// View draws the whole screen.
func (a *App) View() string { return strings.Join(a.Frame(), "\n") }

// Frame is the screen as lines, each exactly as wide as the terminal (Rules L4).
func (a *App) Frame() []string {
	w, h := a.w, a.h
	if w <= 0 || h <= 0 {
		return nil
	}
	if a.tooSmall() {
		return a.tinyScreen(w, h)
	}
	// The splash takes the whole screen: nothing of the screen is drawn (Rules S).
	if a.splashOn {
		lines, _ := a.shown(a.splashFrame, w, h)
		return Pad2(lines, w, h)
	}
	lines := a.base()
	if a.top() != nil {
		lines = DimLines(lines) // everything under the top popup (Rules F8)
	}
	top := a.top()
	under, open := slices.Index(a.stack, top), 0
	for i, l := range a.stack {
		// A popup's layer counts the open ones under it: one closing is not
		// a layer any more, and a target opening while its source closes
		// has its own layer's colour from its first frame (components/color,
		// Rules T1).
		color := LayerColor(open + 1)
		if !l.closing {
			open++
		} else if l.shade.Set {
			color = l.shade
		}
		full := fitRows(l.view.draw(l.w, l.h, color), l.h)
		if i < under {
			full = DimLines(full)
		}
		rows, first, dx := l.drawFrame(full, color)
		Composite(lines, rows, l.x+dx, l.y+first, w)
	}
	if t := a.toast; t != nil {
		full := t.view.draw(t.w, t.h, LayerColor(1))
		rows, first, dx := t.drawFrame(full, LayerColor(1))
		Composite(lines, rows, t.x+dx, t.y+first, w)
	}
	return lines
}

// base is the screen under the popups: the top row, the panels, the footer.
func (a *App) base() []string {
	w, h := a.w, a.h
	if a.Canvas != nil {
		return Pad2(a.Canvas(w, h), w, h)
	}
	lines := make([]string, 0, h)
	multi := len(a.screens) > 1
	switch {
	case multi:
		lines = append(lines, a.chipRow(w))
		if h > 3 {
			lines = append(lines, a.divider(w))
		}
	case a.Status != nil:
		lines = append(lines, Pad(a.Status(w), w))
	}
	areaH := h - len(lines) - 1
	lines = append(lines, a.panelArea(w, areaH)...)
	footer := a.footer(w)
	if a.terminalOpen() {
		// A terminal popup reaches the last row: no footer, its ends blank
		// on either side of the frame (components/dialog/terminal).
		footer = ""
	}
	lines = append(lines, Pad(footer, w))
	return lines[:h]
}

// terminalOpen reports a terminal popup on the stack.
func (a *App) terminalOpen() bool {
	return slices.ContainsFunc(a.stack, func(l *layer) bool {
		_, ok := l.view.(*termView)
		return ok && !l.closing
	})
}

// chipRow is the screen chips, the statusbar on the right (layout/screen).
func (a *App) chipRow(w int) string {
	caps := make([]Capsule, len(a.screens))
	for i, s := range a.screens {
		caps[i] = Capsule{Text: Upper(MarkLabel(s.Key, s.Name)), Selected: i == a.cur}
	}
	status := ""
	if a.Status != nil {
		status = a.Status(max(w/2, 0))
	}
	chips := drawChain(caps, Blue, max(w-Width(status)-1, 1))
	gap := w - Width(chips) - Width(status)
	if gap < 1 {
		return Pad(chips, w)
	}
	return chips + Repeat(" ", gap) + status
}

// divider is the heavy line under the chips; a long job fills it with Green
// from the left.
func (a *App) divider(w int) string {
	if a.Progress != nil {
		if frac, ok := a.Progress(); ok {
			n := min(max(int(frac*float64(w)+0.5), 0), w)
			return Fg(Green).Render(Repeat("━", n)) + Fg(Overlay0).Render(Repeat("━", w-n))
		}
	}
	return Fg(Overlay0).Render(Repeat("━", w))
}

func (a *App) narrow(s *Screen) bool {
	if a.ForceNarrow {
		return true
	}
	return s.NarrowBelow > 0 && a.w < min(s.NarrowBelow, 80)
}

// panelArea draws the current screen's panels in w × h.
func (a *App) panelArea(w, h int) []string {
	area := blankLines(w, h)
	s := a.Screen()
	if s == nil || len(s.Panels) == 0 || h <= 0 {
		return area
	}
	if a.narrow(s) {
		return Pad2(s.Focused().draw(s.at()+1, w, h, true), w, h)
	}
	var rects []Rect
	if s.Layout != nil {
		rects = s.Layout(w, h)
	} else {
		x := 0
		for i := range s.Panels {
			pw := w / len(s.Panels)
			if i == len(s.Panels)-1 {
				pw = w - x
			}
			rects = append(rects, Rect{X: x, Y: 0, W: pw, H: h})
			x += pw
		}
	}
	for i, p := range s.Panels {
		if i >= len(rects) {
			break
		}
		r := rects[i]
		if fp, ok := p.Body.(*FilePreview); ok {
			fp.hide = a.HideSecretsInPreview // the app's setting, as a file-picker's preview has it
		}
		Composite(area, p.draw(i+1, r.W, r.H, i == s.at()), r.X, r.Y, w)
	}
	return area
}

// Pad2 makes lines exactly w × h.
func Pad2(lines []string, w, h int) []string {
	h = max(h, 0) // a popup shorter than its frame has no body rows
	out := make([]string, h)
	for i := range h {
		if i < len(lines) {
			out[i] = Pad(lines[i], w)
		} else {
			out[i] = Repeat(" ", w)
		}
	}
	return out
}

// footer is the bottom row (layout/screen, footer): what can be pressed now.
func (a *App) footer(w int) string {
	return renderHints(a.footerHints(), w, Fg(Blue), Overlay0)
}

func (a *App) footerHints() []KeyHint {
	s := a.Screen()
	if s == nil || s.Focused() == nil {
		return []KeyHint{{"q", "quit"}}
	}
	p := s.Focused()
	if t, ok := p.Body.(*Terminal); ok && t.inside {
		return t.hint() // the exit key and the app's own (components/layout/screen)
	}
	switch {
	case p.typing():
		if !p.Body.HasItem() { // nothing matches: Enter does nothing (components/layout/screen)
			return []KeyHint{{"Tab", "list"}, {"Esc", "clear"}}
		}
		return []KeyHint{{"Enter|Tab", "list"}, {"Esc", "clear"}}
	case p.mode != nil:
		out := []KeyHint{{"?", "help"}}
		for _, k := range p.mode.Keys {
			if !k.NoFooter {
				out = append(out, KeyHint{k.Key, k.Desc})
			}
		}
		return append(out, KeyHint{"Esc", cmpOr(p.mode.Exit, "leave")})
	case len(s.Panels) == 1:
		return []KeyHint{{"Space", "menu"}, {"?", "help"}, {"q", "quit"}}
	}
	return []KeyHint{{"Space", "menu"}, {"?", "help"}, {"Tab|1–" + itoa(len(s.Panels)), "panels"}, {"q", "quit"}}
}
