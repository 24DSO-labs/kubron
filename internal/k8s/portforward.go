package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type PortForwardSession struct {
	ID        string `json:"id"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	LocalPort int    `json:"localPort"`
	PodPort   int    `json:"podPort"`
	Status    string `json:"status"`
	stopChan  chan struct{}
}

type PortForwardManager struct {
	mu       sync.RWMutex
	sessions map[string]*PortForwardSession
	k8sMgr   *Manager
	storeDir string
}

func NewPortForwardManager(k8sMgr *Manager) *PortForwardManager {
	pfm := &PortForwardManager{
		sessions: make(map[string]*PortForwardSession),
		k8sMgr:   k8sMgr,
		storeDir: filepath.Join(GetKubronDir(), "portforwards"),
	}
	// Создаем директорию для хранения если она не существует
	os.MkdirAll(pfm.storeDir, 0755)
	// Загружаем сохраненные port forwards
	pfm.loadFromStorage()
	return pfm
}

func (pfm *PortForwardManager) StartPortForward(namespace, pod string, localPort, podPort int) (*PortForwardSession, error) {
	fmt.Printf("[PORT_FORWARD] StartPortForward called: namespace=%s, pod=%s, localPort=%d, podPort=%d\n", namespace, pod, localPort, podPort)

	conn, err := pfm.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	id := fmt.Sprintf("%s-%s-%d-%d", namespace, pod, localPort, podPort)
	fmt.Printf("[PORT_FORWARD] Generated ID: %s\n", id)

	pfm.mu.Lock()
	if existing, ok := pfm.sessions[id]; ok {
		pfm.mu.Unlock()
		return existing, nil
	}
	pfm.mu.Unlock()

	reqURL := conn.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(pod).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(conn.Config)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания transport: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{
		Scheme: reqURL.Scheme,
		Host:   reqURL.Host,
		Path:   reqURL.Path,
	})

	stopChan := make(chan struct{})
	readyChan := make(chan struct{})

	ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}

	fw, err := portforward.New(dialer, ports, stopChan, readyChan, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания port-forward: %w", err)
	}

	session := &PortForwardSession{
		ID:        id,
		Namespace: namespace,
		Pod:       pod,
		LocalPort: localPort,
		PodPort:   podPort,
		Status:    "starting",
		stopChan:  stopChan,
	}

	pfm.mu.Lock()
	pfm.sessions[id] = session
	pfm.mu.Unlock()

	go func() {
		if err := fw.ForwardPorts(); err != nil {
			pfm.mu.Lock()
			session.Status = "error: " + err.Error()
			pfm.mu.Unlock()
		}
	}()

	go func() {
		<-readyChan
		pfm.mu.Lock()
		session.Status = "active"
		pfm.mu.Unlock()
		pfm.saveToStorage()
	}()

	return session, nil
}

func (pfm *PortForwardManager) StopPortForward(id string) error {
	pfm.mu.Lock()
	defer pfm.mu.Unlock()

	session, ok := pfm.sessions[id]
	if !ok {
		return fmt.Errorf("сессия не найдена: %s", id)
	}

	close(session.stopChan)
	delete(pfm.sessions, id)
	pfm.mu.Unlock()
	pfm.saveToStorage()
	pfm.mu.Lock()
	return nil
}

func (pfm *PortForwardManager) ListPortForwards() []PortForwardSession {
	pfm.mu.RLock()
	defer pfm.mu.RUnlock()

	var result []PortForwardSession
	for _, s := range pfm.sessions {
		result = append(result, PortForwardSession{
			ID:        s.ID,
			Namespace: s.Namespace,
			Pod:       s.Pod,
			LocalPort: s.LocalPort,
			PodPort:   s.PodPort,
			Status:    s.Status,
		})
	}
	return result
}

func (pfm *PortForwardManager) saveToStorage() error {
	pfm.mu.RLock()
	var toSave []*PortForwardSession
	for _, s := range pfm.sessions {
		toSave = append(toSave, &PortForwardSession{
			ID:        s.ID,
			Namespace: s.Namespace,
			Pod:       s.Pod,
			LocalPort: s.LocalPort,
			PodPort:   s.PodPort,
			Status:    s.Status,
		})
	}
	pfm.mu.RUnlock()

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(pfm.storeDir, "sessions.json")
	return os.WriteFile(filePath, data, 0644)
}

func (pfm *PortForwardManager) loadFromStorage() {
	filePath := filepath.Join(pfm.storeDir, "sessions.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return // файл не существует или другая ошибка чтения
	}

	var sessions []*PortForwardSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return
	}

	pfm.mu.Lock()
	defer pfm.mu.Unlock()

	for _, s := range sessions {
		// Помечаем все загруженные сессии как "inactive" так как они не были активны до перезагрузки
		s.Status = "inactive"
		s.stopChan = make(chan struct{})
		pfm.sessions[s.ID] = s
	}
}

func (pfm *PortForwardManager) StopAll() {
	pfm.mu.Lock()
	defer pfm.mu.Unlock()

	for id, session := range pfm.sessions {
		close(session.stopChan)
		delete(pfm.sessions, id)
	}
}

func (pfm *PortForwardManager) StartServicePortForward(namespace, serviceName string, localPort, servicePort int) (*PortForwardSession, error) {
	conn, err := pfm.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Находим поды сервиса через selector
	svc, err := conn.Client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("сервис не найден: %w", err)
	}

	if len(svc.Spec.Selector) == 0 {
		return nil, fmt.Errorf("у сервиса %s нет selector", serviceName)
	}

	// Собираем selector в строку
	var selectorParts []string
	for k, v := range svc.Spec.Selector {
		selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	selector := joinSelector(selectorParts)

	// Находим первый Running pod
	pods, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, fmt.Errorf("не удалось найти поды: %w", err)
	}

	var targetPod string
	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			targetPod = pod.Name
			break
		}
	}

	if targetPod == "" {
		return nil, fmt.Errorf("нет Running подов для сервиса %s", serviceName)
	}

	return pfm.StartPortForward(namespace, targetPod, localPort, servicePort)
}

func joinSelector(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += ","
		}
		result += p
	}
	return result
}
