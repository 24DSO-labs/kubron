package main

import (
	"analog/internal/audit"
	"analog/internal/auth"
	"analog/internal/k8s"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	k8sMgr  *k8s.Manager
	pfMgr   *k8s.PortForwardManager
	audit   *audit.Logger
	auth    *auth.Service // nil, если хранилище пользователей не открылось (см. authErr)
	authErr error
}

func NewApp() *App {
	mgr := k8s.NewManager()
	return &App{
		k8sMgr: mgr,
		pfMgr:  k8s.NewPortForwardManager(mgr),
	}
}

// shutdown вызывается Wails при закрытии приложения: гасим все port-forward туннели.
func (a *App) shutdown(_ context.Context) {
	a.pfMgr.StopAll()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Бэкенд сообщает фронтенду об изменениях кластера (informers) вместо постоянного опроса
	a.k8sMgr.SetNotifier(func(event string, data ...any) {
		wailsRuntime.EventsEmit(ctx, event, data...)
	})
	if err := k8s.InitKubronDir(); err != nil {
		println("Error:", err.Error())
	}
	a.initAuth(k8s.GetKubronDir())
	if a.auth != nil {
		go a.auth.RunIdleWatcher(ctx, 10*time.Second)
	}
}

// === Kubron Cluster Management ===

func (a *App) GetClusterStatuses() []k8s.ClusterStatus {
	if a.authz(auth.PermRead) != nil {
		return nil
	}
	return a.k8sMgr.GetClusterStatuses()
}

// PickKubeconfigFile opens a native file dialog and returns the chosen path
// (empty string when the dialog was cancelled).
func (a *App) PickKubeconfigFile() (string, error) {
	if err := a.authz(auth.PermClusterManage); err != nil {
		return "", err
	}
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите kubeconfig файл",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
}

// AddClusterFromFile adds a cluster from a kubeconfig file previously chosen
// with PickKubeconfigFile.
func (a *App) AddClusterFromFile(path, displayName, icon, color string) (*k8s.ClusterMeta, error) {
	if err := a.authz(auth.PermClusterManage, "displayName", displayName); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, errors.New("kubeconfig file is not selected")
	}
	return k8s.AddClusterFromFile(path, displayName, icon, color)
}

func (a *App) AddClusterFromText(kubeconfigText, displayName, icon, color string) (*k8s.ClusterMeta, error) {
	if err := a.authz(auth.PermClusterManage, "displayName", displayName); err != nil {
		return nil, err
	}
	return k8s.AddClusterFromText(kubeconfigText, displayName, icon, color)
}

func (a *App) DeleteCluster(id string) error {
	if err := a.authz(auth.PermClusterManage, "id", id); err != nil {
		return err
	}
	if err := k8s.DeleteCluster(id); err != nil {
		return err
	}
	// Подключение и проброшенные порты удалённого активного кластера больше не нужны.
	if a.k8sMgr.RemoveConnection(id) {
		a.pfMgr.StopAll()
	}
	return nil
}

func (a *App) UpdateCluster(id, displayName, icon, color string) error {
	if err := a.authz(auth.PermClusterManage, "id", id, "displayName", displayName); err != nil {
		return err
	}
	return k8s.UpdateClusterMeta(id, displayName, icon, color)
}

func (a *App) ConnectToCluster(clusterID string) error {
	if err := a.authz(auth.PermRead); err != nil {
		return err
	}
	err := a.k8sMgr.ConnectToCluster(clusterID)
	if err != nil {
		return err
	}
	config, _ := k8s.LoadKubronConfig()
	if config == nil {
		config = &k8s.KubronConfig{}
	}
	config.LastActiveCluster = clusterID
	k8s.SaveKubronConfig(config)
	return a.k8sMgr.TestConnection()
}

func (a *App) GetLastActiveCluster() string {
	if a.authz(auth.PermRead) != nil {
		return ""
	}
	config, err := k8s.LoadKubronConfig()
	if err != nil {
		return ""
	}
	return config.LastActiveCluster
}

func (a *App) GetAppSettings() (*k8s.KubronConfig, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return k8s.LoadKubronConfig()
}

func (a *App) SaveAppSettings(settings *k8s.KubronConfig) error {
	if err := a.authz(auth.PermRead); err != nil {
		return err
	}
	return k8s.SaveKubronConfig(settings)
}

// === Контексты ===

func (a *App) GetActiveContext() string {
	if a.authz(auth.PermRead) != nil {
		return ""
	}
	return a.k8sMgr.GetActiveContext()
}

// === Overview ===

func (a *App) GetClusterOverview() (*k8s.ClusterOverview, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetClusterOverview()
}

func (a *App) GetOverviewHotspots() (*k8s.OverviewHotspots, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetOverviewHotspots()
}

// === Node actions ===

func (a *App) CordonNode(name string, cordon bool) error {
	if err := a.authz(auth.PermOperate, "node", name, "cordon", cordon); err != nil {
		return err
	}
	return a.k8sMgr.CordonNode(name, cordon)
}

func (a *App) DrainNode(name string) error {
	if err := a.authz(auth.PermWrite, "node", name); err != nil {
		return err
	}
	return a.k8sMgr.DrainNode(name)
}

func (a *App) StartNodeShell(name string) (*k8s.NodeShellSession, error) {
	if err := a.authz(auth.PermExec, "node", name); err != nil {
		return nil, err
	}
	return a.k8sMgr.StartNodeShell(name)
}

// === Argo CD ===

func (a *App) SuspendArgoAutoSync(apps []k8s.ArgoAppRef) ([]k8s.ArgoActionResult, error) {
	if err := a.authz(auth.PermWrite, "apps", argoAuditApps(apps)); err != nil {
		return nil, err
	}
	return a.k8sMgr.SuspendArgoAutoSync(apps)
}

func (a *App) ResumeArgoAutoSync(apps []k8s.ArgoAppRef) ([]k8s.ArgoActionResult, error) {
	if err := a.authz(auth.PermWrite, "apps", argoAuditApps(apps)); err != nil {
		return nil, err
	}
	return a.k8sMgr.ResumeArgoAutoSync(apps)
}

func (a *App) DeleteArgoApplication(namespace, name string, cascade bool) error {
	if err := a.authz(auth.PermWrite, "namespace", namespace, "name", name, "cascade", cascade); err != nil {
		return err
	}
	return a.k8sMgr.DeleteArgoApplication(namespace, name, cascade)
}

// ArgoInstalled сообщает, установлен ли Argo CD (CRD Application) в активном кластере.
func (a *App) ArgoInstalled() (bool, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return false, err
	}
	return a.k8sMgr.ArgoInstalled()
}

// GetArgoApplications возвращает Argo CD Application; пустой namespace — из всех namespace.
func (a *App) GetArgoApplications(namespace string) ([]k8s.ArgoApplicationInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoApplications(namespace)
}

// SyncArgoApplication запускает синхронизацию одного или нескольких Application.
// Результат — по каждому приложению отдельно (одна ошибка не отменяет остальные).
func (a *App) SyncArgoApplication(apps []k8s.ArgoAppRef, opts k8s.ArgoSyncOptions) ([]k8s.ArgoActionResult, error) {
	if err := a.authz(auth.PermOperate, "apps", argoAuditApps(apps), "prune", opts.Prune, "force", opts.Force, "dryRun", opts.DryRun, "revision", opts.Revision); err != nil {
		return nil, err
	}
	return a.k8sMgr.SyncArgoApplication(apps, opts)
}

func (a *App) RefreshArgoApplication(apps []k8s.ArgoAppRef, hard bool) ([]k8s.ArgoActionResult, error) {
	if err := a.authz(auth.PermOperate, "apps", argoAuditApps(apps), "hard", hard); err != nil {
		return nil, err
	}
	return a.k8sMgr.RefreshArgoApplication(apps, hard)
}

func (a *App) TerminateArgoOperation(apps []k8s.ArgoAppRef) ([]k8s.ArgoActionResult, error) {
	if err := a.authz(auth.PermOperate, "apps", argoAuditApps(apps)); err != nil {
		return nil, err
	}
	return a.k8sMgr.TerminateArgoOperation(apps)
}

func (a *App) RollbackArgoApplication(namespace, name string, id int64, prune, dryRun bool) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name, "id", id, "prune", prune, "dryRun", dryRun); err != nil {
		return err
	}
	return a.k8sMgr.RollbackArgoApplication(namespace, name, id, prune, dryRun)
}

// GetArgoApplicationSets возвращает Argo CD ApplicationSet; пустой namespace — из всех namespace.
func (a *App) GetArgoApplicationSets(namespace string) ([]k8s.ArgoApplicationSetInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoApplicationSets(namespace)
}

// GetArgoProjects возвращает Argo CD AppProject; пустой namespace — из всех namespace.
func (a *App) GetArgoProjects(namespace string) ([]k8s.ArgoProjectInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoProjects(namespace)
}

// GetArgoRepositories возвращает репозитории Argo CD (санитизировано: без паролей, токенов и ключей).
func (a *App) GetArgoRepositories(namespace string) ([]k8s.ArgoRepositoryInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoRepositories(namespace)
}

// GetArgoClusters возвращает кластеры Argo CD (санитизировано: без config с токенами и сертификатами).
func (a *App) GetArgoClusters(namespace string) ([]k8s.ArgoClusterInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoClusters(namespace)
}

// GetArgoApplicationGraph возвращает граф ресурсов Application (владение, Service → Pod, Ingress → Service).
func (a *App) GetArgoApplicationGraph(namespace, name string) (*k8s.ArgoGraph, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoApplicationGraph(namespace, name)
}

// GetArgoAPIStatus сообщает, задан ли токен Argo CD API для активного кластера (сам токен наружу не отдаётся).
func (a *App) GetArgoAPIStatus() (*k8s.ArgoAPIStatus, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetArgoAPIStatus()
}

// SetArgoAPIToken сохраняет токен Argo CD API для активного кластера.
// Токен нельзя передавать в authz: аргументы попадают в журнал аудита.
func (a *App) SetArgoAPIToken(token string) error {
	if err := a.authz(auth.PermClusterManage); err != nil {
		return err
	}
	return a.k8sMgr.SetArgoAPIToken(token)
}

// GenerateArgoAPIToken получает токен Argo CD API для аккаунта и сохраняет его для активного кластера.
// Логин и пароль Argo CD нужны только для этого вызова: они не сохраняются и не передаются в authz
// (аргументы authz попадают в журнал аудита); токен фронтенду не возвращается.
func (a *App) GenerateArgoAPIToken(account, username, password string, days int) error {
	if err := a.authz(auth.PermClusterManage, "account", account); err != nil {
		return err
	}
	return a.pfMgr.GenerateArgoAPIToken(account, username, password, days)
}

// DeleteArgoAPIToken удаляет токен Argo CD API активного кластера.
func (a *App) DeleteArgoAPIToken() error {
	if err := a.authz(auth.PermWrite); err != nil {
		return err
	}
	return a.k8sMgr.DeleteArgoAPIToken()
}

// GetArgoApplicationDiff сравнивает желаемое (git) и текущее состояние ресурсов Application через Argo CD API.
// Поднимает туннель к argocd-server, поэтому требует права на проброс портов.
func (a *App) GetArgoApplicationDiff(namespace, name string) (*k8s.ArgoDiff, error) {
	if err := a.authz(auth.PermPortForward, "namespace", namespace, "name", name); err != nil {
		return nil, err
	}
	return a.pfMgr.GetArgoApplicationDiff(namespace, name)
}

// OpenArgoUI пробрасывает порт на argocd-server и возвращает URL для браузера.
func (a *App) OpenArgoUI() (*k8s.ArgoUIInfo, error) {
	if err := a.authz(auth.PermPortForward); err != nil {
		return nil, err
	}
	return a.pfMgr.OpenArgoUI()
}

// argoAuditApps — компактная цель действия для журнала: первые приложения и счётчик остальных.
func argoAuditApps(apps []k8s.ArgoAppRef) string {
	const show = 5
	names := make([]string, 0, show)
	for i, ref := range apps {
		if i == show {
			break
		}
		names = append(names, ref.Namespace+"/"+ref.Name)
	}
	s := strings.Join(names, ",")
	if len(apps) > show {
		s += fmt.Sprintf(" +%d", len(apps)-show)
	}
	return s
}

// === Неймспейсы ===

func (a *App) GetNamespaces() ([]k8s.NamespaceInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetNamespaces()
}

// === Поды ===

func (a *App) GetPods(namespace string) ([]k8s.PodInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPods(namespace)
}

func (a *App) GetPodDetail(namespace, name string) (*k8s.PodDetail, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPodDetail(namespace, name)
}

func (a *App) GetPodLogs(namespace, name, container string, lines int64) (string, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return "", err
	}
	return a.k8sMgr.GetPodLogs(namespace, name, container, lines)
}

func (a *App) EvictPod(namespace, name string) error {
	if err := a.authz(auth.PermWrite, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.EvictPod(namespace, name)
}

// === Deployments ===

func (a *App) GetDeployments(namespace string) ([]k8s.DeploymentInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetDeployments(namespace)
}

func (a *App) ScaleDeployment(namespace, name string, replicas int32) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name, "replicas", replicas); err != nil {
		return err
	}
	return a.k8sMgr.ScaleDeployment(namespace, name, replicas)
}

func (a *App) RestartDeployment(namespace, name string) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.RestartDeployment(namespace, name)
}

func (a *App) GetDeploymentRevisions(namespace, name string) ([]k8s.RevisionInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetDeploymentRevisions(namespace, name)
}

func (a *App) RollbackDeployment(namespace, name string, revision int64) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name, "revision", revision); err != nil {
		return err
	}
	return a.k8sMgr.RollbackDeployment(namespace, name, revision)
}

// === StatefulSets ===

func (a *App) GetStatefulSets(namespace string) ([]k8s.StatefulSetInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetStatefulSets(namespace)
}

func (a *App) ScaleStatefulSet(namespace, name string, replicas int32) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name, "replicas", replicas); err != nil {
		return err
	}
	return a.k8sMgr.ScaleStatefulSet(namespace, name, replicas)
}

// === ReplicaSets ===

func (a *App) GetReplicaSets(namespace string) ([]k8s.ReplicaSetInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetReplicaSets(namespace)
}

func (a *App) ScaleReplicaSet(namespace, name string, replicas int32) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name, "replicas", replicas); err != nil {
		return err
	}
	return a.k8sMgr.ScaleReplicaSet(namespace, name, replicas)
}

// === DaemonSets ===

func (a *App) GetDaemonSets(namespace string) ([]k8s.DaemonSetInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetDaemonSets(namespace)
}

// === Jobs ===

func (a *App) GetJobs(namespace string) ([]k8s.JobInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetJobs(namespace)
}

func (a *App) RestartDaemonSet(namespace, name string) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.RestartDaemonSet(namespace, name)
}

func (a *App) RetryJob(namespace, name string) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.RetryJob(namespace, name)
}

// === CronJobs ===

func (a *App) GetCronJobs(namespace string) ([]k8s.CronJobInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetCronJobs(namespace)
}

func (a *App) GetCronJobJobs(namespace, cronJobName string) ([]k8s.JobInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetCronJobJobs(namespace, cronJobName)
}

func (a *App) GetCronJobDetail(namespace, name string) (*k8s.CronJobDetail, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetCronJobDetail(namespace, name)
}

func (a *App) TriggerCronJob(namespace, name string) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.TriggerCronJob(namespace, name)
}

func (a *App) ToggleCronJobSuspend(namespace, name string) (bool, error) {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return false, err
	}
	return a.k8sMgr.ToggleCronJobSuspend(namespace, name)
}

// === Services ===

func (a *App) GetServices(namespace string) ([]k8s.ServiceInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetServices(namespace)
}

func (a *App) GetServiceDetail(namespace, name string) (*k8s.ServiceDetail, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetServiceDetail(namespace, name)
}

// === Ingresses ===

func (a *App) GetIngresses(namespace string) ([]k8s.IngressInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetIngresses(namespace)
}

// === Nodes ===

func (a *App) GetNodes() ([]k8s.NodeInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetNodes()
}

func (a *App) GetNodeDetail(name string) (*k8s.NodeDetail, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetNodeDetail(name)
}

// === Applications ===

func (a *App) GetApplications(namespace string) ([]k8s.ApplicationInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetApplications(namespace)
}

// === Helm ===

func (a *App) GetHelmReleases() ([]k8s.HelmReleaseInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetHelmReleases()
}

func (a *App) GetHelmCharts() ([]k8s.HelmChartInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetHelmCharts()
}

// === Events ===

func (a *App) GetEvents(namespace string) ([]k8s.EventInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetEvents(namespace)
}

// === ConfigMaps ===

func (a *App) GetConfigMaps(namespace string) ([]k8s.ConfigMapInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetConfigMaps(namespace)
}

func (a *App) GetConfigMapDetail(namespace, name string) (*k8s.ConfigMapInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetConfigMapDetail(namespace, name)
}

func (a *App) UpdateConfigMap(namespace, name string, data map[string]string) error {
	if err := a.authz(auth.PermWrite, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.UpdateConfigMap(namespace, name, data)
}

// === Secrets ===

func (a *App) GetSecrets(namespace string) ([]k8s.SecretInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetSecrets(namespace)
}

func (a *App) GetSecretValue(namespace, name, key string) (string, error) {
	if err := a.authz(auth.PermSecretsRead, "namespace", namespace, "name", name, "key", key); err != nil {
		return "", err
	}
	return a.k8sMgr.GetSecretValue(namespace, name, key)
}

func (a *App) GetSecretBase64Value(namespace, name, key string) (string, error) {
	if err := a.authz(auth.PermSecretsRead, "namespace", namespace, "name", name, "key", key); err != nil {
		return "", err
	}
	return a.k8sMgr.GetSecretBase64Value(namespace, name, key)
}

func (a *App) UpdateSecret(namespace, name string, data map[string]string) error {
	if err := a.authz(auth.PermWrite, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.UpdateSecret(namespace, name, data)
}

// === Storage ===

func (a *App) GetPersistentVolumes() ([]k8s.PVInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPersistentVolumes()
}

func (a *App) GetPersistentVolumeClaims(namespace string) ([]k8s.PVCInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPersistentVolumeClaims(namespace)
}

func (a *App) GetStorageClasses() ([]k8s.StorageClassInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetStorageClasses()
}

// === Config (continued) ===

func (a *App) GetLimitRanges(namespace string) ([]k8s.LimitRangeInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetLimitRanges(namespace)
}

func (a *App) GetHPA(namespace string) ([]k8s.HPAInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetHPA(namespace)
}

func (a *App) GetPodDisruptionBudgets(namespace string) ([]k8s.PDBInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPodDisruptionBudgets(namespace)
}

func (a *App) GetPriorityClasses() ([]k8s.PriorityClassInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPriorityClasses()
}

func (a *App) GetRuntimeClasses() ([]k8s.RuntimeClassInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetRuntimeClasses()
}

func (a *App) GetLeases(namespace string) ([]k8s.LeaseInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetLeases(namespace)
}

func (a *App) GetMutatingWebhookConfigurations() ([]k8s.WebhookConfigurationInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetMutatingWebhookConfigurations()
}

func (a *App) GetValidatingWebhookConfigurations() ([]k8s.WebhookConfigurationInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetValidatingWebhookConfigurations()
}

// === Endpoints ===

func (a *App) GetEndpoints(namespace string) ([]k8s.EndpointListInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetEndpoints(namespace)
}

func (a *App) GetEndpointSlices(namespace string) ([]k8s.EndpointSliceInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetEndpointSlices(namespace)
}

// === Network Policies ===

func (a *App) GetNetworkPolicies(namespace string) ([]k8s.NetworkPolicyInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetNetworkPolicies(namespace)
}

// === Ingress Classes ===

func (a *App) GetIngressClasses() ([]k8s.IngressClassInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetIngressClasses()
}

func (a *App) SetDefaultIngressClass(name string, makeDefault bool) error {
	if err := a.authz(auth.PermWrite, "name", name, "default", makeDefault); err != nil {
		return err
	}
	return a.k8sMgr.SetDefaultIngressClass(name, makeDefault)
}

// === Access Control ===

func (a *App) GetServiceAccounts(namespace string) (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetServiceAccounts(namespace)
}

func (a *App) GetRoles(namespace string) (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetRoles(namespace)
}

func (a *App) GetRoleBindings(namespace string) (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetRoleBindings(namespace)
}

func (a *App) GetClusterRoles() (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetClusterRoles()
}

func (a *App) GetClusterRoleBindings() (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetClusterRoleBindings()
}

func (a *App) GetServiceAccountDetail(namespace, name string) (interface{}, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetServiceAccountDetail(namespace, name)
}

func (a *App) RestartStatefulSet(namespace, name string) error {
	if err := a.authz(auth.PermOperate, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.RestartStatefulSet(namespace, name)
}

func (a *App) GetResourceQuotas(namespace string) ([]k8s.ResourceQuotaInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetResourceQuotas(namespace)
}

// === YAML ===

func (a *App) GetResourceYAML(namespace, kind, name string, clean bool) (string, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return "", err
	}
	// YAML секрета содержит его данные, поэтому требует того же права, что и GetSecretValue.
	if strings.EqualFold(strings.TrimSpace(kind), "secret") {
		if err := a.authz(auth.PermSecretsRead); err != nil {
			return "", err
		}
	}
	return a.k8sMgr.GetResourceYAML(namespace, kind, name, clean)
}

func (a *App) ApplyYAML(namespace, kind, name, jsonData string) error {
	if err := a.authz(auth.PermWrite, "namespace", namespace, "kind", kind, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.ApplyYAML(namespace, kind, name, jsonData)
}

// CreateResourceYAML создаёт новый объект из манифеста (кнопка «+» в списках) и возвращает его имя.
// kind — ключ типа (id определения ресурса на фронтенде). namespace — запасное значение,
// если в манифесте metadata.namespace не указан.
func (a *App) CreateResourceYAML(namespace, kind, yamlData string) (string, error) {
	// authz — первый оператор (его проверяет policy_test). namespace и name в журнал берём из самого
	// манифеста: именно их увидит кластер.
	if err := a.authz(auth.PermWrite, "namespace", k8s.ManifestNamespace(yamlData, namespace),
		"kind", kind, "name", k8s.ManifestName(yamlData)); err != nil {
		return "", err
	}
	return a.k8sMgr.CreateResourceYAML(namespace, kind, yamlData)
}

// === Exec ===

// === Metrics ===

func (a *App) GetPodMetrics(namespace string) ([]k8s.PodMetrics, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetPodMetrics(namespace)
}

func (a *App) GetNodeMetrics() ([]k8s.NodeMetrics, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetNodeMetrics()
}

func (a *App) GetMetricsStatus() (*k8s.MetricsStackStatus, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetMetricsStatus()
}

func (a *App) InstallMetricsStack() error {
	if err := a.authz(auth.PermWrite); err != nil {
		return err
	}
	return a.k8sMgr.InstallMetricsStack()
}

func (a *App) UninstallMetricsStack() error {
	if err := a.authz(auth.PermWrite); err != nil {
		return err
	}
	return a.k8sMgr.UninstallMetricsStack()
}

func (a *App) GetClusterMetricsSnapshot() (*k8s.ClusterMetricsSnapshot, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetClusterMetricsSnapshot()
}

func (a *App) QueryPrometheus(query string) (*k8s.PromQueryResult, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.QueryPrometheus(query)
}

func (a *App) QueryPrometheusRange(query string, startUnix, endUnix int64, step string) (*k8s.PromQueryResult, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	start := time.Unix(startUnix, 0)
	end := time.Unix(endUnix, 0)
	return a.k8sMgr.QueryPrometheusRange(query, start, end, step)
}

// === Port Forward ===

func (a *App) StartPortForward(namespace, pod string, localPort, podPort int) (*k8s.PortForwardSession, error) {
	if err := a.authz(auth.PermPortForward, "namespace", namespace, "pod", pod, "localPort", localPort, "podPort", podPort); err != nil {
		return nil, err
	}
	return a.pfMgr.StartPortForward(namespace, pod, localPort, podPort)
}

func (a *App) StartServicePortForward(namespace, service string, localPort, servicePort int) (*k8s.PortForwardSession, error) {
	if err := a.authz(auth.PermPortForward, "namespace", namespace, "service", service, "localPort", localPort, "servicePort", servicePort); err != nil {
		return nil, err
	}
	return a.pfMgr.StartServicePortForward(namespace, service, localPort, servicePort)
}

func (a *App) StopPortForward(id string) error {
	if err := a.authz(auth.PermPortForward, "id", id); err != nil {
		return err
	}
	return a.pfMgr.StopPortForward(id)
}

func (a *App) ListPortForwards() []k8s.PortForwardSession {
	if a.authz(auth.PermRead) != nil {
		return nil
	}
	return a.pfMgr.ListPortForwards()
}

func (a *App) GetFreePort() (int, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return 0, err
	}
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (a *App) OpenBrowserURL(url string) {
	if a.authz(auth.PermRead) != nil {
		return
	}
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

// === Terminal ===

func (a *App) DetectShell(namespace, pod, container string) (string, error) {
	if err := a.authz(auth.PermExec, "namespace", namespace, "pod", pod, "container", container); err != nil {
		return "", err
	}
	return a.k8sMgr.DetectShell(namespace, pod, container)
}

func (a *App) StartTerminalSession(sessionID, namespace, pod, container, shell string, cols, rows uint16) error {
	if err := a.authz(auth.PermExec, "namespace", namespace, "pod", pod, "container", container, "shell", shell); err != nil {
		return err
	}
	return a.k8sMgr.StartTerminalSession(sessionID, namespace, pod, container, shell, cols, rows)
}

func (a *App) WriteTerminal(sessionID string, data []byte) error {
	if err := a.authz(auth.PermExec); err != nil {
		return err
	}
	return a.k8sMgr.WriteTerminal(sessionID, data)
}

func (a *App) ResizeTerminal(sessionID string, cols, rows uint16) error {
	if err := a.authz(auth.PermExec); err != nil {
		return err
	}
	return a.k8sMgr.ResizeTerminal(sessionID, cols, rows)
}

func (a *App) CloseTerminal(sessionID string) error {
	if err := a.authz(auth.PermExec); err != nil {
		return err
	}
	return a.k8sMgr.CloseTerminal(sessionID)
}

// DeleteResource удаляет ресурс любого поддерживаемого типа. kind — ключ типа
// (совпадает с id определения ресурса на фронтенде, например "deployment").
func (a *App) DeleteResource(kind, namespace, name string) error {
	if err := a.authz(auth.PermWrite, "kind", kind, "namespace", namespace, "name", name); err != nil {
		return err
	}
	return a.k8sMgr.DeleteResource(kind, namespace, name)
}

// === Custom Resources ===

func (a *App) GetCRDs() ([]k8s.CRDInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetCRDs()
}

func (a *App) GetCustomResources(group, version, plural, namespace string) ([]k8s.CustomResourceInfo, error) {
	if err := a.authz(auth.PermRead); err != nil {
		return nil, err
	}
	return a.k8sMgr.GetCustomResources(group, version, plural, namespace)
}

// === Streaming ===

func (a *App) StartLogStream(streamID, namespace, pod, container string, tail int64) error {
	if err := a.authz(auth.PermRead); err != nil {
		return err
	}
	return a.k8sMgr.StartLogStream(streamID, namespace, pod, container, tail)
}

func (a *App) StopLogStream(streamID string) error {
	if err := a.authz(auth.PermRead); err != nil {
		return err
	}
	a.k8sMgr.StopLogStream(streamID)
	return nil
}
