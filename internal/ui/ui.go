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

	applyingMaster int32 // guard: suppress checkbox handler during programmatic updates
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

func (mw *mainWindow) onMasterToggled() {
	if atomic.LoadInt32(&mw.applyingMaster) == 1 {
		return
	}
	on := mw.cbMaster.Checked()
	mw.eng.SetMaster(on)
	mw.updateStatus(on)
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

// Run builds the window, wires it to the engine, starts the hook, and runs the
// GUI message loop until the window closes.
func Run(eng *engine.Engine, initialRules []*config.Rule) error {
	model := &ruleModel{rules: initialRules}
	mw := &mainWindow{eng: eng, model: model}

	outputNames := append([]string{"(同触发键)"}, keys.Names...)

	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "TurboKey 按键连发",
		MinSize:  Size{Width: 760, Height: 480},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					CheckBox{
						AssignTo:         &mw.cbMaster,
						Text:             "总开关 (全局热键 F8)",
						OnCheckedChanged: mw.onMasterToggled,
					},
					Label{AssignTo: &mw.lblStatus, Text: "状态: 已停用"},
					HSpacer{},
				},
			},
			Label{Text: "提示: 本工具以管理员权限运行(向游戏注入按键所需)。总开关开启后, 配置的键在所有程序中都会被连发拦截; 不用时按 F8 关闭。"},
			TableView{
				AssignTo: &mw.tv,
				Columns: []TableViewColumn{
					{Title: "名称", Width: 130},
					{Title: "触发键", Width: 90},
					{Title: "输出键", Width: 110},
					{Title: "模式", Width: 80},
					{Title: "间隔(ms)", Width: 90},
					{Title: "启用", Width: 60},
				},
				Model:           model,
				OnItemActivated: mw.onToggleEnabled,
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "名称"},
					LineEdit{AssignTo: &mw.leName, MaxLength: 20, MinSize: Size{Width: 100}},
					Label{Text: "触发键"},
					ComboBox{AssignTo: &mw.cbTrigger, Model: keys.Names, MinSize: Size{Width: 70}},
					Label{Text: "输出键"},
					ComboBox{AssignTo: &mw.cbOutput, Model: outputNames, MinSize: Size{Width: 100}},
					Label{Text: "模式"},
					ComboBox{AssignTo: &mw.cbMode, Model: []string{"按住连发", "开关连发"}, MinSize: Size{Width: 100}},
					Label{Text: "间隔ms"},
					NumberEdit{AssignTo: &mw.neInterval, Decimals: 0, MinSize: Size{Width: 70}},
					PushButton{Text: "添加规则", OnClicked: mw.onAdd},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{Text: "删除选中", OnClicked: mw.onDelete},
					PushButton{Text: "启用/停用选中 (或双击行)", OnClicked: mw.onToggleEnabled},
					HSpacer{},
				},
			},
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

	eng.OnMasterChange = func(on bool) {
		mw.Synchronize(func() {
			atomic.StoreInt32(&mw.applyingMaster, 1)
			mw.cbMaster.SetChecked(on)
			atomic.StoreInt32(&mw.applyingMaster, 0)
			mw.updateStatus(on)
		})
	}

	eng.SetRules(copyRules(initialRules))
	eng.Start()

	mw.Run()
	return nil
}
