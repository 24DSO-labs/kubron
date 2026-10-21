package k8s

// ArgoResourceRef — ресурс, которым управляет Application (status.resources[]).
type ArgoResourceRef struct {
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Status    string `json:"status"` // Synced | OutOfSync | Unknown
	Health    string `json:"health"` // Healthy | Degraded | Progressing | Suspended | Missing | ""
	Message   string `json:"message"`
	Hook      bool   `json:"hook"`
}

// ArgoHistoryEntry — запись status.history[] (один деплой ревизии).
type ArgoHistoryEntry struct {
	ID         int64  `json:"id"`
	Revision   string `json:"revision"`
	DeployedAt string `json:"deployedAt"`
	Source     string `json:"source"` // repoURL@targetRevision (path)
}

// ArgoApplicationInfo — элемент списка и панели деталей Application.
// Секреты репозиториев/кластеров сюда не попадают (санитизация на стороне Go).
type ArgoApplicationInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Project   string `json:"project"`
	Age       string `json:"age"`

	SyncStatus   string `json:"syncStatus"`   // Synced | OutOfSync | Unknown
	HealthStatus string `json:"healthStatus"` // Healthy | Degraded | Progressing | Suspended | Missing | Unknown
	Revision     string `json:"revision"`

	RepoURL        string `json:"repoURL"`
	Path           string `json:"path"`
	Chart          string `json:"chart"`
	TargetRevision string `json:"targetRevision"`

	DestServer    string `json:"destServer"`
	DestName      string `json:"destName"`
	DestNamespace string `json:"destNamespace"`

	AutoSync    bool `json:"autoSync"`
	SelfHeal    bool `json:"selfHeal"`
	Prune       bool `json:"prune"`
	OperationOn bool `json:"operationOn"` // идёт операция (sync/rollback)

	Message   string             `json:"message"`
	Resources []ArgoResourceRef  `json:"resources"`
	History   []ArgoHistoryEntry `json:"history"`

	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Created     string            `json:"created"`
}
