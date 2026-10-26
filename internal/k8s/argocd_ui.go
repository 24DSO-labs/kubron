// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	argoServerSelector = "app.kubernetes.io/name=argocd-server"
	argoDefaultNS      = "argocd"
	argoParamsConfig   = "argocd-cmd-params-cm"
)

// ArgoUIInfo — куда открыть Argo CD UI в браузере.
type ArgoUIInfo struct {
	URL       string `json:"url"`
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	LocalPort int    `json:"localPort"`
	Reused    bool   `json:"reused"` // использована уже запущенная сессия проброса
}

// OpenArgoUI пробрасывает порт на argocd-server и возвращает URL. Повторный вызов
// переиспользует активную сессию, а не плодит туннели.
func (pfm *PortForwardManager) OpenArgoUI() (*ArgoUIInfo, error) {
	conn, err := pfm.k8sMgr.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svc, err := findArgoServerService(ctx, conn.Client)
	if err != nil {
		return nil, err
	}
	insecure := argoServerInsecure(ctx, conn.Client, svc.Namespace)
	port, ok := pickArgoPort(svc.Spec.Ports, insecure)
	if !ok {
		return nil, fmt.Errorf("у сервиса %s/%s нет портов", svc.Namespace, svc.Name)
	}
	scheme := argoScheme(insecure)
	info := func(local int, reused bool) *ArgoUIInfo {
		return &ArgoUIInfo{
			URL: fmt.Sprintf("%s://localhost:%d", scheme, local), Namespace: svc.Namespace,
			Service: svc.Name, LocalPort: local, Reused: reused,
		}
	}

	forwards := pfm.ListPortForwards()
	for i := range forwards {
		s := &forwards[i]
		if s.Status == PortForwardActive && s.Namespace == svc.Namespace && s.Service == svc.Name && s.ServicePort == int(port.Port) {
			return info(s.LocalPort, true), nil
		}
	}
	local, err := freeLocalPort()
	if err != nil {
		return nil, err
	}
	sess, err := pfm.StartServicePortForward(svc.Namespace, svc.Name, local, int(port.Port))
	if err != nil {
		return nil, err
	}
	return info(sess.LocalPort, false), nil
}

// findArgoServerService ищет Service argocd-server по лейблу; при нескольких предпочитает namespace argocd.
func findArgoServerService(ctx context.Context, client kubernetes.Interface) (*corev1.Service, error) {
	list, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{LabelSelector: argoServerSelector})
	if err != nil {
		// нет прав на все namespace: пробуем стандартный
		list, err = client.CoreV1().Services(argoDefaultNS).List(ctx, metav1.ListOptions{LabelSelector: argoServerSelector})
		if err != nil {
			return nil, fmt.Errorf("поиск сервиса argocd-server: %w", err)
		}
	}
	if len(list.Items) == 0 {
		return nil, errors.New("сервис argocd-server не найден (лейбл app.kubernetes.io/name=argocd-server)")
	}
	items := list.Items
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i].Namespace == argoDefaultNS, items[j].Namespace == argoDefaultNS
		if a != b {
			return a
		}
		if items[i].Namespace != items[j].Namespace {
			return items[i].Namespace < items[j].Namespace
		}
		return items[i].Name < items[j].Name
	})
	return &items[0], nil
}

// argoServerInsecure: server.insecure=true в argocd-cmd-params-cm (сервер отдаёт plain HTTP). Ошибки = false.
func argoServerInsecure(ctx context.Context, client kubernetes.Interface, ns string) bool {
	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, argoParamsConfig, metav1.GetOptions{})
	return err == nil && cm.Data["server.insecure"] == "true"
}

func argoScheme(insecure bool) string {
	if insecure {
		return "http"
	}
	return "https"
}

// pickArgoPort выбирает порт сервиса: https (имя или 443), а при insecure — http (имя или 80).
// Порты обычно ведут на один и тот же 8080 пода, поэтому схема определяется режимом сервера, а не портом.
func pickArgoPort(ports []corev1.ServicePort, insecure bool) (corev1.ServicePort, bool) {
	name, num := "https", int32(443)
	if insecure {
		name, num = "http", 80
	}
	for _, p := range ports {
		if p.Name == name {
			return p, true
		}
	}
	for _, p := range ports {
		if p.Port == num {
			return p, true
		}
	}
	if len(ports) > 0 {
		return ports[0], true
	}
	return corev1.ServicePort{}, false
}

func freeLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("нет свободного локального порта: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
