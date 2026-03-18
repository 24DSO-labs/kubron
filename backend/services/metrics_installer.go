package services

import (
"bufio"
"bytes"
"context"
"fmt"
"io"
"os"
"path/filepath"
"runtime"
"strings"
"k8s.io/apimachinery/pkg/api/errors"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
"k8s.io/apimachinery/pkg/runtime/schema"
"k8s.io/apimachinery/pkg/util/yaml"
"k8s.io/client-go/discovery"
"k8s.io/client-go/dynamic"
"k8s.io/client-go/kubernetes"
"k8s.io/client-go/rest"
yamlutil "sigs.k8s.io/yaml"
)
const MetricsNamespace = "kubron-metrics"
const PrometheusURL = "http://prometheus.kubron-metrics.svc:80"

type MetricsStatus struct {
Status        string            json:"status"
Components    map[string]string json:"components"
PrometheusURL string            json:"prometheusUrl,omitempty"
}

type MetricsInstaller struct {
clientset       kubernetes.Interface
dynamicClient   dynamic.Interface
discoveryClient discovery.DiscoveryInterface
restConfig      *rest.Config
manifestPath    string
}

func NewMetricsInstaller(cs kubernetes.Interface, dc dynamic.Interface, disc discovery.DiscoveryInterface, rc *rest.Config) *MetricsInstaller {
return &MetricsInstaller{
clientset:       cs,
dynamicClient:   dc,
discoveryClient: disc,
restConfig:      rc,
manifestPath:    findManifestPath(),
}
}

func findManifestPath() string {
candidates := []string{
filepath.Join("manifests", "metrics-stack.yaml"),
filepath.Join("backend", "manifests", "metrics-stack.yaml"),
}
if ex, err := os.Executable(); err == nil {
candidates = append(candidates, filepath.Join(filepath.Dir(ex), "manifests", "metrics-stack.yaml"))
}
if _, f, _, ok := runtime.Caller(0); ok {
candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(f)), "manifests", "metrics-stack.yaml"))
}
for _, p := range candidates {
if _, err := os.Stat(p); err == nil {
return p
}
}
return filepath.Join("manifests", "metrics-stack.yaml")
}

func (m *MetricsInstaller) loadManifest() ([]byte, error) {
return os.ReadFile(m.manifestPath)
}