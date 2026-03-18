package main

import (
	"context"
	"mole-gui/backend"
	"os"
)

type App struct {
	ctx     context.Context
	analyze *backend.AnalyzeService
	status  *backend.StatusService
	safety  *backend.SafetyService
	clean   *backend.CleanService
}

func NewApp() *App {
	safety := backend.NewSafetyService()
	return &App{
		analyze: backend.NewAnalyzeService(),
		status:  backend.NewStatusService(),
		safety:  safety,
		clean:   backend.NewCleanService(safety),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.clean.SetContext(ctx)
	a.analyze.SetContext(ctx)
}

// --- Analyze ---

func (a *App) ScanDirectory(path string, maxDepth int) (*backend.ScanResult, error) {
	return a.analyze.ScanDirectory(path, maxDepth)
}

func (a *App) CancelScan() {
	a.analyze.CancelScan()
}

func (a *App) GetDirectoryChildren(path string) ([]*backend.FileEntry, error) {
	return a.analyze.GetDirectoryChildren(path)
}

func (a *App) StartAnalyzeScan(path string, maxDepth int, requestId int) {
	a.analyze.StartScan(path, maxDepth, requestId)
}

func (a *App) CancelAnalyzeScan() {
	a.analyze.CancelScan()
}

func (a *App) StartAnalyzeDrill(path string, requestId int) {
	a.analyze.StartDrill(path, requestId)
}

func (a *App) CancelAnalyzeDrill() {
	a.analyze.CancelDrill()
}

func (a *App) PreFetchChildren(paths []string) {
	a.analyze.PreFetchChildren(paths)
}

func (a *App) GetCachedChildren(path string) []*backend.FileEntry {
	items, ok := a.analyze.GetCachedChildren(path)
	if !ok {
		return nil
	}
	return items
}

// --- Status ---

func (a *App) GetSystemStatus() (*backend.SystemStatus, error) {
	return a.status.GetSystemStatus()
}

func (a *App) GetTopProcesses(limit int) ([]backend.ProcessInfo, error) {
	return a.status.GetTopProcesses(limit)
}

// --- Safety ---

func (a *App) MoveToTrash(path string) error {
	return a.safety.MoveToTrash(path)
}

func (a *App) MoveMultipleToTrash(paths []string) ([]string, []string) {
	return a.safety.MoveMultipleToTrash(paths)
}

func (a *App) GetOperationHistory(limit int) ([]backend.OperationLog, error) {
	return a.safety.GetOperationHistory(limit)
}

// --- Clean ---

func (a *App) CleanDryRun() (*backend.CleanResult, error) {
	return a.clean.DryRun()
}

func (a *App) StartDrill(path string, requestId int) {
	a.clean.StartDrill(path, requestId)
}

func (a *App) CancelDrill() {
	a.clean.CancelDrill()
}

func (a *App) ExecuteClean(paths []string) (*backend.CleanExecResult, error) {
	return a.clean.ExecuteClean(paths)
}

func (a *App) PlayTrashSound() {
	a.safety.PlayTrashSound()
}

// --- Utility ---

func (a *App) GetHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}
