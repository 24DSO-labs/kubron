package k8s

// ArgoResourceRef — ресурс, которым управляет Application (status.resources[]).
type ArgoResourceRef struct {
	Group           string `json:"group"`
	Version         string `json:"version"`
	Kind            string `json:"kind"`
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	Status          string `json:"status"` // Synced | OutOfSync | Unknown
	Health          string `json:"health"` // Healthy | Degraded | Progressing | Suspended | Missing | ""
	Message         string `json:"message"`
	Hook            bool   `json:"hook"`
	RequiresPruning bool   `json:"requiresPruning"` // есть в кластере, но удалён из Git: удалится при sync с prune
}

// ArgoHistoryEntry — запись status.history[] (один деплой ревизии).
type ArgoHistoryEntry struct {
	ID         int64  `json:"id"`
	Revision   string `json:"revision"`
	DeployedAt string `json:"deployedAt"`
	Source     string `json:"source"` // repoURL@targetRevision (path)
}

// ArgoSourceInfo — один источник Application (spec.source или элемент spec.sources[]).
// Helm-параметры и values намеренно не включаются: в них могут быть секреты.
type ArgoSourceInfo struct {
	RepoURL        string `json:"repoURL"` // без userinfo (user:token@)
	Path           string `json:"path"`
	Chart          string `json:"chart"`
	TargetRevision string `json:"targetRevision"`
	Ref            string `json:"ref"` // имя ref-источника (multi-source)
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

	LastSyncAt    string `json:"lastSyncAt"`    // status.operationState.finishedAt (RFC3339), "" если синков не было
	LastSyncPhase string `json:"lastSyncPhase"` // Succeeded | Failed | Error | Running | ""

	Message    string             `json:"message"`
	Sources    []ArgoSourceInfo   `json:"sources"` // все источники; RepoURL/Path/... выше — первый из них
	Conditions []ConditionInfo    `json:"conditions"`
	Resources  []ArgoResourceRef  `json:"resources"`
	History    []ArgoHistoryEntry `json:"history"` // новые записи первыми

	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Created     string            `json:"created"`
}
