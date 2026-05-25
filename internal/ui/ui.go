//go:build windows

// Package ui implements the native Win32 GUI (lxn/walk) for managing rules.
package ui

import (
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"turbokey/internal/config"
	"turbokey/internal/engine"
	"turbokey/internal/keys"
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
			return "(同触发键)"
		}
		return keys.Name(r.OutputVK)
	case 3:
		if r.Mode == config.ModeToggle {
			return "开关"
		}
		return "按住"
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
	leName     *walk.LineEdit
	cbTrigger  *walk.ComboBox
	cbOutput   *walk.ComboBox
	cbMode     *walk.ComboBox
	neInterval *walk.NumberEdit

	ni         *walk.NotifyIcon // system tray icon; nil if tray setup failed
	trayMaster *walk.Action     // checkable master toggle in the tray menu

	applyingMaster int32 // guard: suppress checkbox handler during programmatic updates
	exiting        bool  // true once the user chose Exit, so close is not redirected to tray
	toldTray       bool  // whether the "minimized to tray" balloon was already shown
}

func copyRules(in []*config.Rule) []*config.Rule {
	out := make([]*config.Rule, len(in))
	for i, r := range in {
		rc := *r
		out[i] = &rc
	}
	return out
}

// apply pushes the current rules into the engine (as independent copies) and
// persists them to disk.
func (mw *mainWindow) apply() {
	mw.eng.SetRules(copyRules(mw.model.rules))
	config.Save(mw.model.rules)
}

func (mw *mainWindow) updateStatus(on bool) {
	if on {
		mw.lblStatus.SetText("状态: 运行中")
	} else {
		mw.lblStatus.SetText("状态: 已停用")
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
		mw.ni.ShowInfo("TurboKey", "已最小化到系统托盘。右键托盘图标可退出。")
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
	ni.SetToolTip("TurboKey 按键连发")

	showAct := walk.NewAction()
	showAct.SetText("显示主界面")
	showAct.Triggered().Attach(mw.restore)
	ni.ContextMenu().Actions().Add(showAct)

	mw.trayMaster = walk.NewAction()
	mw.trayMaster.SetText("启用连发 (F8)")
	mw.trayMaster.SetCheckable(true)
	mw.trayMaster.Triggered().Attach(func() { mw.applyMaster(mw.trayMaster.Checked()) })
	ni.ContextMenu().Actions().Add(mw.trayMaster)

	exitAct := walk.NewAction()
	exitAct.SetText("退出")
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
func Run(eng *engine.Engine, initialRules []*config.Rule) error {
	model := &ruleModel{rules: initialRules}
	mw := &mainWindow{eng: eng, model: model}

	outputNames := append([]string{"(同触发键)"}, keys.Names...)

	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "TurboKey 按键连发",
		MinSize:  Size{Width: 400, Height: 340},
		Size:     Size{Width: 430, Height: 410},
		Layout:   VBox{Spacing: 6},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					CheckBox{
						AssignTo:         &mw.cbMaster,
						Text:             "总开关 (F8)",
						OnCheckedChanged: mw.onMasterToggled,
					},
					Label{AssignTo: &mw.lblStatus, Text: "状态: 已停用"},
					HSpacer{},
				},
			},
			TableView{
				AssignTo: &mw.tv,
				Columns: []TableViewColumn{
					{Title: "名称", Width: 84},
					{Title: "触发", Width: 52},
					{Title: "输出", Width: 72},
					{Title: "模式", Width: 50},
					{Title: "间隔", Width: 52},
					{Title: "启用", Width: 40},
				},
				Model:           model,
				OnItemActivated: mw.onToggleEnabled,
			},
			Composite{
				Layout: Grid{Columns: 2, Spacing: 6, MarginsZero: true},
				Children: []Widget{
					Label{Text: "名称"},
					LineEdit{AssignTo: &mw.leName, MaxLength: 20},
					Label{Text: "触发键"},
					ComboBox{AssignTo: &mw.cbTrigger, Model: keys.Names},
					Label{Text: "输出键"},
					ComboBox{AssignTo: &mw.cbOutput, Model: outputNames},
					Label{Text: "模式"},
					ComboBox{AssignTo: &mw.cbMode, Model: []string{"按住连发", "开关连发"}},
					Label{Text: "间隔(ms)"},
					NumberEdit{AssignTo: &mw.neInterval, Decimals: 0},
				},
			},
			Composite{
				Layout: HBox{Spacing: 6, MarginsZero: true},
				Children: []Widget{
					PushButton{Text: "添加", OnClicked: mw.onAdd},
					PushButton{Text: "删除选中", OnClicked: mw.onDelete},
					PushButton{Text: "启用/停用", OnClicked: mw.onToggleEnabled},
				},
			},
			Label{Text: "双击行可启停 · 关闭窗口即最小化到托盘"},
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

	eng.SetRules(copyRules(initialRules))
	eng.Start()

	mw.Run()
	return nil
}
