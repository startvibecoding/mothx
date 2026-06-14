package tui

import "time"

// resetTranscriptState clears rendered conversation bookkeeping without
// changing session/provider configuration.
func (a *App) resetTranscriptState() {
	a.messages = nil
	a.toolResults = nil
	a.liveContent = ""
	a.viewport.SetContent("")
	a.viewport.GotoBottom()
	a.currentPlan = nil
	a.assistantRaw = make(map[int]string)
	a.assistantRendered = make(map[int]string)
	a.assistantDirty = make(map[int]bool)
	a.thinkRaw = make(map[int]string)
	a.printedMessageIdx = make(map[int]bool)
	a.currentAssistantIdx = -1
	a.currentThinkIdx = -1
	a.closeToolModal()
}

func (a *App) clearQueuedInput() {
	a.inputQueueMu.Lock()
	a.inputQueue = a.inputQueue[:0]
	a.lastInputTime = time.Time{}
	a.inputQueueMu.Unlock()
}
