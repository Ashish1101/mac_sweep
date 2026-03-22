package main

import (
	"context"
	"macsweep/backend"
	"os"
)

type App struct {
	ctx         context.Context
	analyze     *backend.AnalyzeService
	status      *backend.StatusService
	safety      *backend.SafetyService
	clean       *backend.CleanService
	permissions *backend.PermissionsService
}

func NewApp() *App {
	safety := backend.NewSafetyService()
	return &App{
		analyze:     backend.NewAnalyzeService(),
		status:      backend.NewStatusService(),
		safety:      safety,
		clean:       backend.NewCleanService(safety),
		permissions: backend.NewPermissionsService(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.clean.SetContext(ctx)
	a.analyze.SetContext(ctx)
}

// --- Analyze ---

func (a *App) CancelScan() {
	a.analyze.CancelScan()
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

func (a *App) GetAllProcesses() ([]backend.ProcessInfo, error) {
	return a.status.GetAllProcesses()
}

func (a *App) KillProcess(pid int) backend.KillResult {
	return a.status.KillProcess(pid)
}

func (a *App) GetCPUDetail() backend.CPUDetail {
	return a.status.GetCPUDetail()
}

func (a *App) GetMemoryDetail() backend.MemoryDetail {
	return a.status.GetMemoryDetail()
}

func (a *App) GetDiskDetail() backend.DiskDetail {
	return a.status.GetDiskDetail()
}

func (a *App) GetBatteryDetail() backend.BatteryDetail {
	return a.status.GetBatteryDetail()
}

func (a *App) GetNetworkDetail() backend.NetworkDetail {
	return a.status.GetNetworkDetail()
}

func (a *App) GetWiFiDetail() backend.WiFiDetail {
	return a.status.GetWiFiDetail()
}

func (a *App) GetWiFiPassword(networkName string) (string, error) {
	return a.status.GetWiFiPassword(networkName)
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

func (a *App) RestoreFromTrash(originalPath string) error {
	return a.safety.RestoreFromTrash(originalPath)
}

func (a *App) RestoreAllFromTrash() backend.RestoreResult {
	return a.safety.RestoreAllFromTrash()
}

func (a *App) CanAccessTrash() bool {
	return a.safety.CanAccessTrash()
}

func (a *App) GetTrashItems() []string {
	return a.safety.GetTrashItems()
}

func (a *App) EmptyTrash() error {
	return a.safety.EmptyTrash()
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

func (a *App) PlaySound(soundID string) {
	a.safety.PlaySound(soundID)
}

// --- Permissions ---

func (a *App) CheckFullDiskAccess() backend.FDAStatus {
	return a.permissions.CheckFullDiskAccess()
}

func (a *App) OpenFullDiskAccessSettings() error {
	return a.permissions.OpenFullDiskAccessSettings()
}

// --- Utility ---

func (a *App) GetHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}
