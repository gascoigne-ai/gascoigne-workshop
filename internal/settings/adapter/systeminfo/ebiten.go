package systeminfo

import "github.com/hajimehoshi/ebiten/v2"

// Ebiten is the graphics source of the running game: call it only while
// ebiten.RunGame runs, where the graphics library is known.
func Ebiten() Graphics {
	var debugInfo ebiten.DebugInfo
	ebiten.ReadDebugInfo(&debugInfo)
	monitor := ebiten.Monitor()
	g := Graphics{
		Library:       debugInfo.GraphicsLibrary.String(),
		GPUImageBytes: debugInfo.TotalGPUImageMemoryUsageInBytes,
		Vsync:         ebiten.IsVsyncEnabled(),
		FPS:           ebiten.ActualFPS(),
		TPS:           ebiten.ActualTPS(),
	}
	if monitor != nil {
		g.Monitor, g.DeviceScale = monitor.Name(), monitor.DeviceScaleFactor()
		g.MonitorW, g.MonitorH = monitor.Size()
	}
	g.WindowW, g.WindowH = ebiten.WindowSize()
	return g
}
