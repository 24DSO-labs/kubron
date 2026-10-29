// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// Статусы сессии проброса портов.
const (
	PortForwardStarting = "starting"
	PortForwardActive   = "active"
	PortForwardError    = "error"
)

// portForwardReadyTimeout — сколько ждём установления туннеля при старте.
const portForwardReadyTimeout = 15 * time.Second

// PortForwardSession — сессия проброса порта в под (напрямую или через сервис).
type PortForwardSession struct {
	ID string `json:"id"`
	// ClusterID — кластер, в котором открыт туннель. Туннель построен на подключении
	// этого кластера и после переключения активного кластера продолжает вести в него.
	ClusterID string `json:"clusterId,omitempty"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Service   string `json:"service,omitempty"` // заполнено, если проброс запущен через сервис
	// ServicePort — порт сервиса, выбранный пользователем (podPort — разрешённый targetPort).
	ServicePort int    `json:"servicePort,omitempty"`
	LocalPort   int    `json:"localPort"`
	PodPort     int    `json:"podPort"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Name        string `json:"name"`   // = ID, нужен списку ресурсов для ключа и открытия панели
	Target      string `json:"target"` // "pod/nginx" или "svc/nginx:80"

	stopChan chan struct{}
	stopOnce sync.Once
}

func (s *PortForwardSession) stop() {
	s.stopOnce.Do(func() { close(s.stopChan) })
}

// snapshot возвращает копию без внутреннего состояния (вызывать под pfm.mu).
func (s *PortForwardSession) snapshot() PortForwardSession {
	return PortForwardSession{
		ID:          s.ID,
		ClusterID:   s.ClusterID,
		Namespace:   s.Namespace,
		Pod:         s.Pod,
		Service:     s.Service,
		ServicePort: s.ServicePort,
		LocalPort:   s.LocalPort,
		PodPort:     s.PodPort,
		Status:      s.Status,
		Error:       s.Error,
		Name:        s.Name,
		Target:      s.Target,
	}
}

// PortForwardManager управляет сессиями проброса портов. Сессии живут только
// пока работает приложение: туннель нельзя восстановить после перезапуска.
type PortForwardManager struct {
	mu       sync.Mutex
	sessions map[string]*PortForwardSession
	k8sMgr   *Manager
}

// NewSessionPortForwardManager — менеджер port-forward вида сессии (без очистки каталога данных).
func NewSessionPortForwardManager(k8sMgr *Manager) *PortForwardManager {
	return &PortForwardManager{sessions: make(map[string]*PortForwardSession), k8sMgr: k8sMgr}
}

func NewPortForwardManager(k8sMgr *Manager) *PortForwardManager {
	// Ранние версии сохраняли сессии на диск и после перезапуска показывали
	// «мёртвые» записи. Убираем оставшийся файл.
	_ = os.RemoveAll(filepath.Join(GetKubronDir(), "portforwards"))
	return &PortForwardManager{
		sessions: make(map[string]*PortForwardSession),
		k8sMgr:   k8sMgr,
	}
}

// StartPortForward пробрасывает localPort на podPort указанного пода.
func (pfm *PortForwardManager) StartPortForward(namespace, pod string, localPort, podPort int) (*PortForwardSession, error) {
	id := fmt.Sprintf("%s-%s-%d-%d", namespace, pod, localPort, podPort)
	return pfm.start(id, namespace, pod, "", 0, localPort, podPort)
}

// StartServicePortForward пробрасывает localPort на порт сервиса: выбирается готовый
// под сервиса, а targetPort (в том числе именованный) разрешается в номер порта пода.
func (pfm *PortForwardManager) StartServicePortForward(namespace, serviceName string, localPort, servicePort int) (*PortForwardSession, error) {
	conn, err := pfm.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := conn.Client.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}
	if len(svc.Spec.Selector) == 0 {
		return nil, fmt.Errorf("service %s has no selector", serviceName)
	}

	var svcPort *corev1.ServicePort
	for i := range svc.Spec.Ports {
		if int(svc.Spec.Ports[i].Port) == servicePort {
			svcPort = &svc.Spec.Ports[i]
			break
		}
	}
	if svcPort == nil {
		return nil, fmt.Errorf("service %s has no port %d", serviceName, servicePort)
	}

	pods, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find pods: %w", err)
	}

	pod := pickReadyPod(pods.Items)
	if pod == nil {
		return nil, fmt.Errorf("no ready pods for service %s", serviceName)
	}

	podPort, err := resolveTargetPort(svcPort, pod)
	if err != nil {
		return nil, err
	}

	id := fmt.Sprintf("%s-svc-%s-%d-%d", namespace, serviceName, localPort, servicePort)
	return pfm.start(id, namespace, pod.Name, serviceName, servicePort, localPort, podPort)
}

// pickReadyPod возвращает первый под в фазе Running с условием Ready.
func pickReadyPod(pods []corev1.Pod) *corev1.Pod {
	for i := range pods {
		p := &pods[i]
		if p.Status.Phase != corev1.PodRunning || p.DeletionTimestamp != nil {
			continue
		}
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				return p
			}
		}
	}
	return nil
}

// resolveTargetPort переводит targetPort сервиса в номер порта контейнера пода.
func resolveTargetPort(sp *corev1.ServicePort, pod *corev1.Pod) (int, error) {
	tp := sp.TargetPort
	if tp.String() == "" || tp.String() == "0" {
		return int(sp.Port), nil // targetPort не задан — совпадает с port
	}
	if tp.IntVal != 0 && tp.StrVal == "" {
		return int(tp.IntVal), nil
	}
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			if cp.Name == tp.StrVal {
				return int(cp.ContainerPort), nil
			}
		}
	}
	return 0, fmt.Errorf("port %q not found in containers of pod %s", tp.StrVal, pod.Name)
}

func (pfm *PortForwardManager) start(id, namespace, pod, service string, servicePort, localPort, podPort int) (*PortForwardSession, error) {
	conn, err := pfm.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	target := "pod/" + pod
	if service != "" {
		target = fmt.Sprintf("svc/%s:%d", service, servicePort)
	}

	session := &PortForwardSession{
		ID:          id,
		ClusterID:   conn.Name,
		Namespace:   namespace,
		Pod:         pod,
		Service:     service,
		ServicePort: servicePort,
		LocalPort:   localPort,
		PodPort:     podPort,
		Status:      PortForwardStarting,
		stopChan:    make(chan struct{}),
		Name:        id,
		Target:      target,
	}

	// Проверка и вставка выполняются под одним локом, чтобы два одновременных
	// вызова не создали два туннеля на один порт.
	pfm.mu.Lock()
	if existing, ok := pfm.sessions[id]; ok {
		if existing.Status != PortForwardError {
			snap := existing.snapshot()
			pfm.mu.Unlock()
			return &snap, nil
		}
		existing.stop()
		delete(pfm.sessions, id)
	}
	if max := pfm.k8sMgr.Limits().PortForwards; max > 0 && len(pfm.sessions) >= max {
		pfm.mu.Unlock()
		return nil, fmt.Errorf("too many port-forwards (maximum %d)", max)
	}
	pfm.sessions[id] = session
	pfm.mu.Unlock()

	fail := func(err error) (*PortForwardSession, error) {
		pfm.mu.Lock()
		if pfm.sessions[id] == session {
			delete(pfm.sessions, id)
		}
		pfm.mu.Unlock()
		session.stop()
		return nil, err
	}

	reqURL := conn.Client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(pod).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(conn.Config)
	if err != nil {
		return fail(fmt.Errorf("failed to create transport: %w", err))
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{
		Scheme: reqURL.Scheme,
		Host:   reqURL.Host,
		Path:   reqURL.Path,
	})

	readyChan := make(chan struct{})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", localPort, podPort)}, session.stopChan, readyChan, nil, nil)
	if err != nil {
		return fail(fmt.Errorf("failed to create port-forward: %w", err))
	}

	done := make(chan error, 1)
	go func() {
		err := fw.ForwardPorts()
		done <- err

		// Туннель закрылся: если сессию не остановили вручную, отмечаем ошибку,
		// чтобы интерфейс не показывал «живую» сессию.
		pfm.mu.Lock()
		defer pfm.mu.Unlock()
		if pfm.sessions[id] != session {
			return
		}
		msg := "connection closed"
		if err != nil {
			msg = err.Error()
		}
		session.Status = PortForwardError
		session.Error = msg
	}()

	select {
	case <-readyChan:
		pfm.mu.Lock()
		if pfm.sessions[id] == session && session.Status == PortForwardStarting {
			session.Status = PortForwardActive
		}
		snap := session.snapshot()
		pfm.mu.Unlock()
		return &snap, nil
	case err := <-done:
		if err == nil {
			err = fmt.Errorf("connection closed before becoming ready")
		}
		return fail(fmt.Errorf("failed to start port-forward: %w", err))
	case <-time.After(portForwardReadyTimeout):
		return fail(fmt.Errorf("timed out establishing port-forward (%s)", portForwardReadyTimeout))
	}
}

// StopPortForward останавливает сессию и удаляет её из списка.
func (pfm *PortForwardManager) StopPortForward(id string) error {
	pfm.mu.Lock()
	session, ok := pfm.sessions[id]
	if ok {
		delete(pfm.sessions, id)
	}
	pfm.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	session.stop()
	return nil
}

// ListPortForwards возвращает копии всех сессий, отсортированные по ID.
func (pfm *PortForwardManager) ListPortForwards() []PortForwardSession {
	pfm.mu.Lock()
	defer pfm.mu.Unlock()

	result := make([]PortForwardSession, 0, len(pfm.sessions))
	for _, s := range pfm.sessions {
		result = append(result, s.snapshot())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// StopAll останавливает все сессии (выход из учётной записи, закрытие приложения).
func (pfm *PortForwardManager) StopAll() {
	pfm.mu.Lock()
	sessions := make([]*PortForwardSession, 0, len(pfm.sessions))
	for id, s := range pfm.sessions {
		sessions = append(sessions, s)
		delete(pfm.sessions, id)
	}
	pfm.mu.Unlock()

	for _, s := range sessions {
		s.stop()
	}
}
