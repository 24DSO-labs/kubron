package main

import (
	"analog/internal/k8s"
	"context"
	"net"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	k8s.InitKubronDir()
}

// === Kubron Cluster Management ===

func (a *App) GetClusterStatuses() []k8s.ClusterStatus {
	return a.k8sMgr.GetClusterStatuses()
}

func (a App) AddClusterFromFile(displayName, icon, color string) (*k8s.ClusterMeta, error) {
	path, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите kubeconfig файл",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "All Files", Pattern: "."},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	return k8s.AddClusterFromFile(path, displayName, icon, color)
}

func (a *App) AddClusterFromText(kubeconfigText, displayName, icon, color string) (*k8s.ClusterMeta, error) {
	return k8s.AddClusterFromText(kubeconfigText, displayName, icon, color)
}

func (a *App) DeleteCluster(id string) error {
	return k8s.DeleteCluster(id)
}

func (a *App) UpdateCluster(id, displayName, icon, color string) error {
	return k8s.UpdateClusterMeta(id, displayName, icon, color)
}

func (a *App) ConnectToCluster(clusterID string) error {
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
	config, err := k8s.LoadKubronConfig()
	if err != nil {
		return ""
	}
	return config.LastActiveCluster
}

func (a *App) GetAppSettings() (*k8s.KubronConfig, error) {
	return k8s.LoadKubronConfig()
}

func (a *App) SaveAppSettings(settings *k8s.KubronConfig) error {
	return k8s.SaveKubronConfig(settings)
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

func (a *App) EvictPod(namespace, name string) error {
	return a.k8sMgr.EvictPod(namespace, name)
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

func (a *App) DeleteDeployment(namespace, name string) error {
	return a.k8sMgr.DeleteDeployment(namespace, name)
}

// === StatefulSets ===

func (a *App) GetStatefulSets(namespace string) ([]k8s.StatefulSetInfo, error) {
	return a.k8sMgr.GetStatefulSets(namespace)
}

func (a *App) ScaleStatefulSet(namespace, name string, replicas int32) error {
	return a.k8sMgr.ScaleStatefulSet(namespace, name, replicas)
}

// === ReplicaSets ===

func (a *App) GetReplicaSets(namespace string) ([]k8s.ReplicaSetInfo, error) {
	return a.k8sMgr.GetReplicaSets(namespace)
}

func (a *App) ScaleReplicaSet(namespace, name string, replicas int32) error {
	return a.k8sMgr.ScaleReplicaSet(namespace, name, replicas)
}

// === DaemonSets ===

func (a *App) GetDaemonSets(namespace string) ([]k8s.DaemonSetInfo, error) {
	return a.k8sMgr.GetDaemonSets(namespace)
}

// === Jobs ===

func (a *App) GetJobs(namespace string) ([]k8s.JobInfo, error) {
	return a.k8sMgr.GetJobs(namespace)
}

func (a *App) DeleteJob(namespace, name string) error {
	return a.k8sMgr.DeleteJob(namespace, name)
}

func (a *App) DeleteDaemonSet(namespace, name string) error {
	return a.k8sMgr.DeleteDaemonSet(namespace, name)
}

func (a *App) RestartDaemonSet(namespace, name string) error {
	return a.k8sMgr.RestartDaemonSet(namespace, name)
}

func (a *App) DeleteReplicaSet(namespace, name string) error {
	return a.k8sMgr.DeleteReplicaSet(namespace, name)
}

func (a *App) DeleteConfigMap(namespace, name string) error {
	return a.k8sMgr.DeleteConfigMap(namespace, name)
}

func (a *App) RetryJob(namespace, name string) error {
	return a.k8sMgr.RetryJob(namespace, name)
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

func (a *App) GetNodeDetail(name string) (*k8s.NodeDetail, error) {
	return a.k8sMgr.GetNodeDetail(name)
}

// === Helm ===

func (a *App) GetHelmReleases() ([]k8s.HelmReleaseInfo, error) {
	return a.k8sMgr.GetHelmReleases()
}

func (a *App) GetHelmCharts() ([]k8s.HelmChartInfo, error) {
	return a.k8sMgr.GetHelmCharts()
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

func (a *App) GetSecretValue(namespace, name, key string) (string, error) {
	return a.k8sMgr.GetSecretValue(namespace, name, key)
}

// === Storage ===

func (a *App) GetPersistentVolumes() ([]k8s.PVInfo, error) {
	return a.k8sMgr.GetPersistentVolumes()
}

func (a *App) GetPersistentVolumeClaims(namespace string) ([]k8s.PVCInfo, error) {
	return a.k8sMgr.GetPersistentVolumeClaims(namespace)
}

func (a *App) GetStorageClasses() ([]k8s.StorageClassInfo, error) {
	return a.k8sMgr.GetStorageClasses()
}

// === Config (continued) ===

func (a *App) GetLimitRanges(namespace string) ([]k8s.LimitRangeInfo, error) {
	return a.k8sMgr.GetLimitRanges(namespace)
}

func (a *App) GetHPA(namespace string) ([]k8s.HPAInfo, error) {
	return a.k8sMgr.GetHPA(namespace)
}

func (a *App) GetPodDisruptionBudgets(namespace string) ([]k8s.PDBInfo, error) {
	return a.k8sMgr.GetPodDisruptionBudgets(namespace)
}

func (a *App) GetPriorityClasses() ([]k8s.PriorityClassInfo, error) {
	return a.k8sMgr.GetPriorityClasses()
}

func (a *App) GetRuntimeClasses() ([]k8s.RuntimeClassInfo, error) {
	return a.k8sMgr.GetRuntimeClasses()
}

func (a *App) GetLeases(namespace string) ([]k8s.LeaseInfo, error) {
	return a.k8sMgr.GetLeases(namespace)
}

// === Endpoints ===

func (a *App) GetEndpoints(namespace string) ([]k8s.EndpointListInfo, error) {
	return a.k8sMgr.GetEndpoints(namespace)
}

// === Network Policies ===

func (a *App) GetNetworkPolicies(namespace string) ([]k8s.NetworkPolicyInfo, error) {
	return a.k8sMgr.GetNetworkPolicies(namespace)
}

// === Ingress Classes ===

func (a *App) GetIngressClasses() ([]k8s.IngressClassInfo, error) {
	return a.k8sMgr.GetIngressClasses()
}

// === Access Control ===

func (a *App) GetServiceAccounts(namespace string) (interface{}, error) {
	conn, err := a.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (a *App) GetRoles(namespace string) (interface{}, error) {
	conn, err := a.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (a *App) GetRoleBindings(namespace string) (interface{}, error) {
	conn, err := a.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (a *App) GetClusterRoles() (interface{}, error) {
	conn, err := a.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (a *App) GetClusterRoleBindings() (interface{}, error) {
	conn, err := a.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (a *App) GetServiceAccountDetail(namespace, name string) (interface{}, error) {
	return a.k8sMgr.GetServiceAccountDetail(namespace, name)
}

func (a *App) DeleteService(namespace, name string) error {
	return a.k8sMgr.DeleteService(namespace, name)
}

func (a *App) DeleteSecret(namespace, name string) error {
	return a.k8sMgr.DeleteSecret(namespace, name)
}

func (a *App) DeleteStatefulSet(namespace, name string) error {
	return a.k8sMgr.DeleteStatefulSet(namespace, name)
}

func (a *App) RestartStatefulSet(namespace, name string) error {
	return a.k8sMgr.RestartStatefulSet(namespace, name)
}

func (a *App) DeleteServiceAccount(namespace, name string) error {
	return a.k8sMgr.DeleteServiceAccount(namespace, name)
}

func (a *App) DeleteRole(namespace, name string) error {
	return a.k8sMgr.DeleteRole(namespace, name)
}

func (a *App) DeleteRoleBinding(namespace, name string) error {
	return a.k8sMgr.DeleteRoleBinding(namespace, name)
}

func (a *App) DeleteClusterRole(name string) error {
	return a.k8sMgr.DeleteClusterRole(name)
}

func (a *App) DeleteClusterRoleBinding(name string) error {
	return a.k8sMgr.DeleteClusterRoleBinding(name)
}

func (a *App) GetResourceQuotas(namespace string) ([]k8s.ResourceQuotaInfo, error) {
	return a.k8sMgr.GetResourceQuotas(namespace)
}

func (a *App) DeleteCronJob(namespace, name string) error {
	return a.k8sMgr.DeleteCronJob(namespace, name)
}

func (a *App) DeleteIngress(namespace, name string) error {
	return a.k8sMgr.DeleteIngress(namespace, name)
}

func (a *App) DeleteIngressClass(name string) error {
	return a.k8sMgr.DeleteIngressClass(name)
}

func (a *App) DeleteNetworkPolicy(namespace, name string) error {
	return a.k8sMgr.DeleteNetworkPolicy(namespace, name)
}

func (a *App) DeletePersistentVolume(name string) error {
	return a.k8sMgr.DeletePersistentVolume(name)
}

func (a *App) DeletePersistentVolumeClaim(namespace, name string) error {
	return a.k8sMgr.DeletePersistentVolumeClaim(namespace, name)
}

func (a *App) DeleteStorageClass(name string) error {
	return a.k8sMgr.DeleteStorageClass(name)
}

func (a *App) DeleteResourceQuota(namespace, name string) error {
	return a.k8sMgr.DeleteResourceQuota(namespace, name)
}

func (a *App) DeleteLimitRange(namespace, name string) error {
	return a.k8sMgr.DeleteLimitRange(namespace, name)
}

func (a *App) DeleteHPA(namespace, name string) error {
	return a.k8sMgr.DeleteHPA(namespace, name)
}

func (a *App) DeletePDB(namespace, name string) error {
	return a.k8sMgr.DeletePDB(namespace, name)
}

func (a *App) DeletePriorityClass(name string) error {
	return a.k8sMgr.DeletePriorityClass(name)
}

func (a *App) DeleteRuntimeClass(name string) error {
	return a.k8sMgr.DeleteRuntimeClass(name)
}

func (a *App) DeleteLease(namespace, name string) error {
	return a.k8sMgr.DeleteLease(namespace, name)
}

// === YAML ===

func (a *App) GetResourceYAML(namespace, kind, name string, clean bool) (string, error) {
	return a.k8sMgr.GetResourceYAML(namespace, kind, name, clean)
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

func (a *App) GetMetricsStatus() (*k8s.MetricsStackStatus, error) {
	return a.k8sMgr.GetMetricsStatus()
}

func (a *App) InstallMetricsStack() error {
	return a.k8sMgr.InstallMetricsStack()
}

func (a *App) UninstallMetricsStack() error {
	return a.k8sMgr.UninstallMetricsStack()
}

func (a *App) GetClusterMetricsSnapshot() (*k8s.ClusterMetricsSnapshot, error) {
	return a.k8sMgr.GetClusterMetricsSnapshot()
}

func (a *App) GetNodeMetricsByRole(role string) (*k8s.NodeMetricsByRole, error) {
	return a.k8sMgr.GetNodeMetricsByRole(role)
}

func (a *App) QueryPrometheus(query string) (*k8s.PromQueryResult, error) {
	return a.k8sMgr.QueryPrometheus(query)
}

func (a *App) QueryPrometheusRange(query string, startUnix, endUnix int64, step string) (*k8s.PromQueryResult, error) {
	start := time.Unix(startUnix, 0)
	end := time.Unix(endUnix, 0)
	return a.k8sMgr.QueryPrometheusRange(query, start, end, step)
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

func (a *App) GetFreePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func (a *App) OpenBrowserURL(url string) {
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

// === Terminal ===

func (a *App) DetectShell(namespace, pod, container string) (string, error) {
	return a.k8sMgr.DetectShell(namespace, pod, container)
}

func (a *App) StartTerminalSession(sessionID, namespace, pod, container, shell string, cols, rows uint16) error {
	return a.k8sMgr.StartTerminalSession(sessionID, namespace, pod, container, shell, cols, rows)
}

func (a *App) WriteTerminal(sessionID string, data []byte) error {
	return a.k8sMgr.WriteTerminal(sessionID, data)
}

func (a *App) ReadTerminal(sessionID string) ([]byte, error) {
	return a.k8sMgr.ReadTerminalBytes(sessionID)
}

func (a *App) ResizeTerminal(sessionID string, cols, rows uint16) error {
	return a.k8sMgr.ResizeTerminal(sessionID, cols, rows)
}

func (a *App) CloseTerminal(sessionID string) error {
	return a.k8sMgr.CloseTerminal(sessionID)
}
