package boot

import (
	"os"
	"sync/atomic"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── 系统托盘：最小化到托盘的窗口关闭行为 ─────────────────────────────
//
// 生命周期约束（energye/systray v1.0.3 实测源码结论）：
//   - Quit() 内部走 sync.Once，一个进程只能停一次托盘；
//   - Windows 上 Register() 不启动消息泵，必须用 RunWithExternalLoop
//     返回的 start/end 自管消息循环。
// 因此采用「至多启动一次」策略：
//   - 启动时若设置已开启 → 立即注册托盘；
//   - 运行中首次开启该设置 → 此时注册（之后关闭设置仅改变关闭行为，图标保留至进程退出）；
//   - 进程退出（OnShutdown）→ end() 摘除图标并停止消息泵。
//
// quitRequested 是退出绕行标志：wails v2 的 runtime.Quit 会被 OnBeforeClose
// 拦截（desktop/windows/frontend.go v2.12 Quit() 实测），托盘菜单「退出」
// 与应用菜单「退出」都必须先置位再 Quit，否则开启本功能后无法退出。

var (
	trayStarted   atomic.Bool
	trayEndFn     func()
	trayIconBytes []byte
	quitRequested atomic.Bool
)

// requestQuit 置位退出标志并真正退出应用（托盘「退出」与菜单「退出」共用）。
func requestQuit() {
	quitRequested.Store(true)
	if appCtx != nil {
		wailsRuntime.Quit(appCtx)
	} else {
		os.Exit(0)
	}
}

// showMainWindow 从托盘恢复主窗口（含最小化状态的还原）。
func showMainWindow() {
	if appCtx == nil {
		return
	}
	wailsRuntime.WindowUnminimise(appCtx)
	wailsRuntime.WindowShow(appCtx)
}

// buildTrayMenu 构建托盘右键菜单（语言切换时经 ResetMenu 后重建）。
func buildTrayMenu() {
	showItem := systray.AddMenuItem(T("tray.show"), T("tray.show"))
	showItem.Click(func() { showMainWindow() })
	systray.AddSeparator()
	quitItem := systray.AddMenuItem(T("tray.quit"), T("tray.quit"))
	quitItem.Click(func() { requestQuit() })
}

// ensureTray 注册托盘（幂等：进程内至多注册一次）。
func ensureTray() {
	if !trayStarted.CompareAndSwap(false, true) {
		return
	}
	start, end := systray.RunWithExternalLoop(func() {
		if len(trayIconBytes) > 0 {
			systray.SetIcon(trayIconBytes)
		}
		systray.SetTooltip("Gridea Pro")
		// 左键单击：显示主窗口；右键：弹出菜单
		systray.SetOnClick(func(menu systray.IMenu) { showMainWindow() })
		systray.SetOnRClick(func(menu systray.IMenu) { menu.ShowMenu() })
		buildTrayMenu()
	}, func() {})
	trayEndFn = end
	start()
}

// rebuildTrayMenu 语言切换后重建托盘菜单文案（未注册时为空操作）。
func rebuildTrayMenu() {
	if !trayStarted.Load() {
		return
	}
	systray.ResetMenu()
	buildTrayMenu()
}

// stopTray 应用退出前摘除托盘图标并停止消息泵。
func stopTray() {
	if trayEndFn != nil {
		trayEndFn()
	}
}
