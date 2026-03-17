package main

import (
	"analog/internal/k8s"
	"context"
)

type App struct {
	ctx    context.Context
	k8sMgr *k8s.Manager
	pfMgr  *k8s.PortForwardManager
}

func NewApp() *App {
	mgr := k8s.NewManager()
	return &App{
		k8sMgr: mgr,
		pfMgr:  k8s.NewPortForwardManager(mgr),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// === Контексты ===

func (a *App) GetContexts() ([]k8s.ContextInfo, error) {
	return a.k8sMgr.LoadContexts()
}

func (a *App) ConnectToContext(contextName string) error {
	err := a.k8sMgr.ConnectToContext(contextName)
	if err != nil {
		return err
	}
	return a.k8sMgr.TestConnection()
}

func (a *App) GetActiveContext() string {
	return a.k8sMgr.GetActiveContext()
}

// === Overview ===

func (a *App) GetClusterOverview() (*k8s.ClusterOverview, error) {
	return a.k8sMgr.GetClusterOverview()
}

// === Неймспейсы ===

func (a *App) GetNamespaces() ([]k8s.NamespaceInfo, error) {
	return a.k8sMgr.GetNamespaces()
}

// === Поды ===

func (a *App) GetPods(namespace string) ([]k8s.PodInfo, error) {
	return a.k8sMgr.GetPods(namespace)
}

func (a *App) GetPodDetail(namespace, name string) (*k8s.PodDetail, error) {
	return a.k8sMgr.GetPodDetail(namespace, name)
}

func (a *App) GetPodLogs(namespace, name, container string, lines int64) (string, error) {
	return a.k8sMgr.GetPodLogs(namespace, name, container, lines)
}

func (a *App) DeletePod(namespace, name string) error {
	return a.k8sMgr.DeletePod(namespace, name)
}

// === Deployments ===

func (a *App) GetDeployments(namespace string) ([]k8s.DeploymentInfo, error) {
	return a.k8sMgr.GetDeployments(namespace)
}

func (a *App) ScaleDeployment(namespace, name string, replicas int32) error {
	return a.k8sMgr.ScaleDeployment(namespace, name, replicas)
}

func (a *App) RestartDeployment(namespace, name string) error {
	return a.k8sMgr.RestartDeployment(namespace, name)
}

func (a *App) GetDeploymentRevisions(namespace, name string) ([]k8s.RevisionInfo, error) {
	return a.k8sMgr.GetDeploymentRevisions(namespace, name)
}

func (a *App) RollbackDeployment(namespace, name string, revision int64) error {
	return a.k8sMgr.RollbackDeployment(namespace, name, revision)
}

// === StatefulSets ===

func (a *App) GetStatefulSets(namespace string) ([]k8s.StatefulSetInfo, error) {
	return a.k8sMgr.GetStatefulSets(namespace)
}

// === DaemonSets ===

func (a *App) GetDaemonSets(namespace string) ([]k8s.DaemonSetInfo, error) {
	return a.k8sMgr.GetDaemonSets(namespace)
}

// === Jobs ===

func (a *App) GetJobs(namespace string) ([]k8s.JobInfo, error) {
	return a.k8sMgr.GetJobs(namespace)
}

// === CronJobs ===

func (a *App) GetCronJobs(namespace string) ([]k8s.CronJobInfo, error) {
	return a.k8sMgr.GetCronJobs(namespace)
}

// === Services ===

func (a *App) GetServices(namespace string) ([]k8s.ServiceInfo, error) {
	return a.k8sMgr.GetServices(namespace)
}

func (a *App) GetServiceDetail(namespace, name string) (*k8s.ServiceDetail, error) {
	return a.k8sMgr.GetServiceDetail(namespace, name)
}

// === Ingresses ===

func (a *App) GetIngresses(namespace string) ([]k8s.IngressInfo, error) {
	return a.k8sMgr.GetIngresses(namespace)
}

// === Nodes ===

func (a *App) GetNodes() ([]k8s.NodeInfo, error) {
	return a.k8sMgr.GetNodes()
}

// === Events ===

func (a *App) GetEvents(namespace string) ([]k8s.EventInfo, error) {
	return a.k8sMgr.GetEvents(namespace)
}

// === ConfigMaps ===

func (a *App) GetConfigMaps(namespace string) ([]k8s.ConfigMapInfo, error) {
	return a.k8sMgr.GetConfigMaps(namespace)
}

func (a *App) GetConfigMapDetail(namespace, name string) (*k8s.ConfigMapInfo, error) {
	return a.k8sMgr.GetConfigMapDetail(namespace, name)
}

// === Secrets ===

func (a *App) GetSecrets(namespace string) ([]k8s.SecretInfo, error) {
	return a.k8sMgr.GetSecrets(namespace)
}

// === Storage ===

func (a *App) GetPersistentVolumes() ([]k8s.PVInfo, error) {
	return a.k8sMgr.GetPersistentVolumes()
}

func (a *App) GetPersistentVolumeClaims(namespace string) ([]k8s.PVCInfo, error) {
	return a.k8sMgr.GetPersistentVolumeClaims(namespace)
}

// === YAML ===

func (a *App) GetResourceYAML(namespace, kind, name string) (string, error) {
	return a.k8sMgr.GetResourceYAML(namespace, kind, name)
}

func (a *App) ApplyYAML(namespace, kind, name, jsonData string) error {
	return a.k8sMgr.ApplyYAML(namespace, kind, name, jsonData)
}

// === Exec ===

func (a *App) ExecCommand(namespace, pod, container, command string) (string, error) {
	return a.k8sMgr.ExecCommand(namespace, pod, container, command)
}

// === Metrics ===

func (a *App) GetPodMetrics(namespace string) ([]k8s.PodMetrics, error) {
	return a.k8sMgr.GetPodMetrics(namespace)
}

func (a *App) GetNodeMetrics() ([]k8s.NodeMetrics, error) {
	return a.k8sMgr.GetNodeMetrics()
}

// === Port Forward ===

func (a *App) StartPortForward(namespace, pod string, localPort, podPort int) (*k8s.PortForwardSession, error) {
	return a.pfMgr.StartPortForward(namespace, pod, localPort, podPort)
}

func (a *App) StartServicePortForward(namespace, service string, localPort, servicePort int) (*k8s.PortForwardSession, error) {
	return a.pfMgr.StartServicePortForward(namespace, service, localPort, servicePort)
}

func (a *App) StopPortForward(id string) error {
	return a.pfMgr.StopPortForward(id)
}

func (a *App) ListPortForwards() []k8s.PortForwardSession {
	return a.pfMgr.ListPortForwards()
}

// === Terminal ===

func (a *App) DetectShell(namespace, pod, container string) (string, error) {
	return a.k8sMgr.DetectShell(namespace, pod, container)
}

func (a *App) ExecTerminal(namespace, pod, container, shell, command string) (*k8s.TerminalOutput, error) {
	return a.k8sMgr.ExecTerminal(namespace, pod, container, shell, command)
}

func (a *App) TabComplete(namespace, pod, container, shell, partial string) ([]string, error) {
	return a.k8sMgr.TabComplete(namespace, pod, container, shell, partial)
}
