//go:build windows

// Package ui implements the native Win32 GUI (lxn/walk) for managing rules.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"turbokey/internal/config"
	"turbokey/internal/engine"
	"turbokey/internal/i18n"
	"turbokey/internal/keys"
	"turbokey/internal/winput"
)

// ruleModel adapts the rule slice to a walk TableView.
type ruleModel struct {
	walk.TableModelBase
	rules []*config.Rule
}

func (m *ruleModel) RowCount() int { return len(m.rules) }

func (m *ruleModel) Value(row, col int) interface{} {
	r := m.rules[row]
	switch col {
	case 0:
		return r.Name
	case 1:
		return keys.Name(r.TriggerVK)
	case 2:
		if r.OutputVK == 0 {
			return i18n.T("output.same")
		}
		return keys.Name(r.OutputVK)
	case 3:
		if r.Mode == config.ModeToggle {
			return i18n.T("mode.toggle")
		}
		return i18n.T("mode.hold")
	case 4:
		return r.IntervalMs
	case 5:
		if r.Enabled {
			return "✓"
		}
		return "✗"
	}
	return ""
}

type mainWindow struct {
	*walk.MainWindow
	eng   *engine.Engine
	model *ruleModel

	tv         *walk.TableView
	cbMaster   *walk.CheckBox
	lblStatus  *walk.Label
	cbLang     *walk.ComboBox
	leTargets  *walk.LineEdit
	btnCapture *walk.PushButton
	leName     *walk.LineEdit
	cbTrigger  *walk.ComboBox
	cbOutput   *walk.ComboBox
	cbMode     *walk.ComboBox
	neInterval *walk.NumberEdit

	ni         *walk.NotifyIcon // system tray icon; nil if tray setup failed
	trayMaster *walk.Action     // checkable master toggle in the tray menu

	lang           string   // selected language code: "" (auto) | "zh" | "en"
	targets        []string // process exe names the tool acts in (empty = all)
	langReady      bool     // true once the language combo's initial value is set
	applyingMaster int32  // guard: suppress checkbox handler during programmatic updates
	exiting        bool   // true once the user chose Exit, so close is not redirected to tray
	toldTray       bool   // whether the "minimized to tray" balloon was already shown
}

func copyRules(in []*config.Rule) []*config.Rule {
	out := make([]*config.Rule, len(in))
	for i, r := range in {
		rc := *r
		out[i] = &rc
	}
	return out
}

// save writes the full current configuration to disk.
func (mw *mainWindow) save() {
	config.Save(&config.File{Lang: mw.lang, Targets: mw.targets, Rules: mw.model.rules})
}

// apply pushes the current rules into the engine (as independent copies) and
// persists them to disk.
func (mw *mainWindow) apply() {
	mw.eng.SetRules(copyRules(mw.model.rules))
	mw.save()
}

func parseTargets(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// applyTargets reads the target field, pushes it to the engine, and persists it.
func (mw *mainWindow) applyTargets() {
	mw.targets = parseTargets(mw.leTargets.Text())
	mw.eng.SetTargets(mw.targets)
	mw.save()
}

// onCapture grabs the foreground app after a short countdown (giving the user
// time to switch to it) and adds its process name to the target list.
func (mw *mainWindow) onCapture() {
	go func() {
		for i := 3; i >= 1; i-- {
			n := i
			mw.Synchronize(func() { mw.btnCapture.SetText(fmt.Sprintf("%d…", n)) })
			time.Sleep(900 * time.Millisecond)
		}
		name := winput.ForegroundProcessName()
		mw.Synchronize(func() {
			mw.btnCapture.SetText(i18n.T("btn.capture"))
			if name == "" || strings.EqualFold(name, "turbokey.exe") {
				return
			}
			for _, c := range parseTargets(mw.leTargets.Text()) {
				if strings.EqualFold(c, name) {
					return // already listed
				}
			}
			cur := parseTargets(mw.leTargets.Text())
			cur = append(cur, name)
			mw.leTargets.SetText(strings.Join(cur, ", "))
			mw.applyTargets()
		})
	}()
}

func (mw *mainWindow) updateStatus(on bool) {
	if on {
		mw.lblStatus.SetText(i18n.T("status.on"))
	} else {
		mw.lblStatus.SetText(i18n.T("status.off"))
	}
}

// applyMaster sets the engine master state and reflects it across all controls.
func (mw *mainWindow) applyMaster(on bool) {
	mw.eng.SetMaster(on)
	mw.syncMasterUI(on)
}

// syncMasterUI reflects the master state on the checkbox, tray action, and status
// label without re-entering the engine. The guard suppresses the checkbox's own
// change handler while it is set programmatically.
func (mw *mainWindow) syncMasterUI(on bool) {
	atomic.StoreInt32(&mw.applyingMaster, 1)
	mw.cbMaster.SetChecked(on)
	if mw.trayMaster != nil {
		mw.trayMaster.SetChecked(on)
	}
	atomic.StoreInt32(&mw.applyingMaster, 0)
	mw.updateStatus(on)
}

func (mw *mainWindow) onMasterToggled() {
	if atomic.LoadInt32(&mw.applyingMaster) == 1 {
		return
	}
	mw.applyMaster(mw.cbMaster.Checked())
}

func (mw *mainWindow) onAdd() {
	tvk, ok := keys.VK(mw.cbTrigger.Text())
	if !ok {
		return
	}
	var ovk uint16
	if mw.cbOutput.CurrentIndex() > 0 {
		ovk, _ = keys.VK(mw.cbOutput.Text())
	}
	mode := config.ModeHold
	if mw.cbMode.CurrentIndex() == 1 {
		mode = config.ModeToggle
	}
	interval := int(mw.neInterval.Value())
	if interval < 1 {
		interval = 1
	}
	mw.model.rules = append(mw.model.rules, &config.Rule{
		Name:       mw.leName.Text(),
		TriggerVK:  tvk,
		OutputVK:   ovk,
		Mode:       mode,
		IntervalMs: interval,
		Enabled:    true,
	})
	mw.model.PublishRowsReset()
	mw.apply()
}

func (mw *mainWindow) onDelete() {
	i := mw.tv.CurrentIndex()
	if i < 0 || i >= len(mw.model.rules) {
		return
	}
	mw.model.rules = append(mw.model.rules[:i], mw.model.rules[i+1:]...)
	mw.model.PublishRowsReset()
	mw.apply()
}

func (mw *mainWindow) onToggleEnabled() {
	i := mw.tv.CurrentIndex()
	if i < 0 || i >= len(mw.model.rules) {
		return
	}
	mw.model.rules[i].Enabled = !mw.model.rules[i].Enabled
	mw.model.PublishRowsReset()
	mw.apply()
}

// onLangChanged persists the chosen language and relaunches so the whole UI is
// rebuilt in it. Ignored during the initial programmatic selection.
func (mw *mainWindow) onLangChanged() {
	if !mw.langReady {
		return
	}
	code := i18n.CodeAt(mw.cbLang.CurrentIndex())
	if code == mw.lang {
		return
	}
	mw.lang = code
	mw.save()
	mw.relaunch()
}

// relaunch starts a fresh instance (inheriting this elevated process's rights,
// so no new UAC prompt) and exits the current one.
func (mw *mainWindow) relaunch() {
	if exe, err := os.Executable(); err == nil {
		cmd := exec.Command(exe)
		cmd.Dir = filepath.Dir(exe)
		cmd.Start()
	}
	mw.exit()
}

// restore shows and activates the main window (e.g. from the tray).
func (mw *mainWindow) restore() {
	mw.Show()
	mw.Activate()
}

// hideToTray hides the window, keeping the app running in the tray.
func (mw *mainWindow) hideToTray() {
	mw.Hide()
	if !mw.toldTray && mw.ni != nil {
		mw.toldTray = true
		mw.ni.ShowInfo(i18n.T("tray.balloon.title"), i18n.T("tray.balloon.body"))
	}
}

// exit quits the application for real (the tray menu's Exit).
func (mw *mainWindow) exit() {
	mw.exiting = true
	if mw.ni != nil {
		mw.ni.Dispose()
	}
	mw.Close()
}

// setupTray installs the system-tray icon and its context menu. On failure the
// app simply runs without a tray (mw.ni stays nil).
func (mw *mainWindow) setupTray() {
	ni, err := walk.NewNotifyIcon(mw.MainWindow)
	if err != nil {
		return
	}
	mw.ni = ni
	ni.SetIcon(walk.IconApplication())
	ni.SetToolTip(i18n.T("tray.tooltip"))

	showAct := walk.NewAction()
	showAct.SetText(i18n.T("tray.show"))
	showAct.Triggered().Attach(mw.restore)
	ni.ContextMenu().Actions().Add(showAct)

	mw.trayMaster = walk.NewAction()
	mw.trayMaster.SetText(i18n.T("tray.master"))
	mw.trayMaster.SetCheckable(true)
	mw.trayMaster.Triggered().Attach(func() { mw.applyMaster(mw.trayMaster.Checked()) })
	ni.ContextMenu().Actions().Add(mw.trayMaster)

	exitAct := walk.NewAction()
	exitAct.SetText(i18n.T("tray.exit"))
	exitAct.Triggered().Attach(mw.exit)
	ni.ContextMenu().Actions().Add(exitAct)

	// Left-click the tray icon to bring the window back.
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			mw.restore()
		}
	})

	ni.SetVisible(true)
}

// Run builds the window, wires it to the engine, starts the hook, and runs the
// GUI message loop until the window closes.
func Run(eng *engine.Engine, cfg *config.File) error {
	model := &ruleModel{rules: cfg.Rules}
	mw := &mainWindow{eng: eng, model: model, lang: cfg.Lang, targets: cfg.Targets}

	outputNames := append([]string{i18n.T("output.same")}, keys.Names...)

	if err := (MainWindow{
		AssignTo:           &mw.MainWindow,
		Title:              i18n.T("app.title"),
		RightToLeftLayout:  i18n.IsRTL(), // mirror the whole layout for RTL languages
		RightToLeftReading: i18n.IsRTL(),
		MinSize:            Size{Width: 400, Height: 376},
		Size:               Size{Width: 440, Height: 448},
		Layout:             VBox{Spacing: 6},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					CheckBox{
						AssignTo:         &mw.cbMaster,
						Text:             i18n.T("master"),
						OnCheckedChanged: mw.onMasterToggled,
					},
					Label{AssignTo: &mw.lblStatus, Text: i18n.T("status.off")},
					HSpacer{},
					Label{Text: i18n.T("lbl.lang")},
					ComboBox{
						AssignTo:              &mw.cbLang,
						Model:                 i18n.DisplayNames(),
						OnCurrentIndexChanged: mw.onLangChanged,
						MinSize:               Size{Width: 110},
					},
				},
			},
			Composite{
				Layout: HBox{MarginsZero: true, Spacing: 6},
				Children: []Widget{
					Label{Text: i18n.T("lbl.scope")},
					LineEdit{AssignTo: &mw.leTargets, OnEditingFinished: mw.applyTargets},
					PushButton{AssignTo: &mw.btnCapture, Text: i18n.T("btn.capture"), OnClicked: mw.onCapture, MaxSize: Size{Width: 120}},
				},
			},
			TableView{
				AssignTo: &mw.tv,
				Columns: []TableViewColumn{
					{Title: i18n.T("col.name"), Width: 84},
					{Title: i18n.T("col.trigger"), Width: 52},
					{Title: i18n.T("col.output"), Width: 72},
					{Title: i18n.T("col.mode"), Width: 50},
					{Title: i18n.T("col.interval"), Width: 52},
					{Title: i18n.T("col.enabled"), Width: 40},
				},
				Model:           model,
				OnItemActivated: mw.onToggleEnabled,
			},
			Composite{
				Layout: Grid{Columns: 2, Spacing: 6, MarginsZero: true},
				Children: []Widget{
					Label{Text: i18n.T("lbl.name")},
					LineEdit{AssignTo: &mw.leName, MaxLength: 20},
					Label{Text: i18n.T("lbl.trigger")},
					ComboBox{AssignTo: &mw.cbTrigger, Model: keys.Names},
					Label{Text: i18n.T("lbl.output")},
					ComboBox{AssignTo: &mw.cbOutput, Model: outputNames},
					Label{Text: i18n.T("lbl.mode")},
					ComboBox{AssignTo: &mw.cbMode, Model: []string{i18n.T("mode.hold"), i18n.T("mode.toggle")}},
					Label{Text: i18n.T("lbl.interval")},
					NumberEdit{AssignTo: &mw.neInterval, Decimals: 0},
				},
			},
			Composite{
				Layout: HBox{Spacing: 6, MarginsZero: true},
				Children: []Widget{
					PushButton{Text: i18n.T("btn.add"), OnClicked: mw.onAdd},
					PushButton{Text: i18n.T("btn.delete"), OnClicked: mw.onDelete},
					PushButton{Text: i18n.T("btn.toggle"), OnClicked: mw.onToggleEnabled},
				},
			},
			Label{Text: i18n.T("footer")},
		},
	}).Create(); err != nil {
		return err
	}

	// Initialize widget values after creation: walk's declarative builder applies
	// properties in random (map) order, so range-checked properties like
	// NumberEdit.Value or ComboBox.CurrentIndex must be set here instead.
	mw.neInterval.SetRange(1, 100000)
	mw.neInterval.SetValue(10)
	mw.cbTrigger.SetCurrentIndex(0)
	mw.cbOutput.SetCurrentIndex(0)
	mw.cbMode.SetCurrentIndex(0)
	mw.cbLang.SetCurrentIndex(i18n.IndexOf(mw.lang))
	mw.langReady = true // enable the language handler only after the initial value
	mw.leTargets.SetText(strings.Join(mw.targets, ", "))

	mw.SetIcon(walk.IconApplication())

	// Reflect master changes made via the global F8 hotkey.
	eng.OnMasterChange = func(on bool) {
		mw.Synchronize(func() { mw.syncMasterUI(on) })
	}

	// System tray: closing the window hides it there instead of quitting; Exit is
	// available from the tray menu.
	mw.setupTray()
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if mw.ni != nil && !mw.exiting {
			*canceled = true
			mw.hideToTray()
		}
	})

	eng.SetTargets(cfg.Targets)
	eng.SetRules(copyRules(cfg.Rules))
	eng.Start()

	mw.Run()
	return nil
}
