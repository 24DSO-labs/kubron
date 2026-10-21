package k8s

import (
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// objectMeta — общие поля панели деталей (Age, Created, Labels, Annotations, Finalizers).
// Wails не разворачивает встроенные структуры в TS-модели надёжно, поэтому поля
// продублированы в каждом *Info, а заполняются одним вызовом.
type objectMeta struct {
	Age         string
	Created     string
	Labels      map[string]string
	Annotations map[string]string
	Finalizers  []string
}

func metaOf(o metav1.ObjectMeta) objectMeta {
	return objectMeta{
		Age:         formatAge(o.CreationTimestamp.Time),
		Created:     o.CreationTimestamp.Format(time.RFC3339),
		Labels:      o.Labels,
		Annotations: o.Annotations,
		Finalizers:  o.Finalizers,
	}
}

// TODO: дубль formatSelector с networkpolicies
// formatSelector — selector в одну строку: "a=1, b=2, key In (x, y)". Порядок детерминирован.
func formatSelectorcm(sel *metav1.LabelSelector) string {
	if sel == nil {
		return ""
	}
	parts := make([]string, 0, len(sel.MatchLabels)+len(sel.MatchExpressions))
	for k, v := range sel.MatchLabels {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	for _, e := range sel.MatchExpressions {
		if len(e.Values) > 0 {
			parts = append(parts, fmt.Sprintf("%s %s (%s)", e.Key, e.Operator, strings.Join(e.Values, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", e.Key, e.Operator))
		}
	}
	return strings.Join(parts, ", ")
}

// labelSelectorInfo — структурный selector для секции Selector в панели.
func labelSelectorInfo(sel *metav1.LabelSelector) *PVCSelectorInfo {
	if sel == nil {
		return nil
	}
	out := &PVCSelectorInfo{MatchLabels: sel.MatchLabels, MatchExpressions: []MatchExpression{}}
	for _, e := range sel.MatchExpressions {
		out.MatchExpressions = append(out.MatchExpressions, MatchExpression{
			Key:      e.Key,
			Operator: string(e.Operator),
			Values:   e.Values,
		})
	}
	return out
}

func formatTime(t metav1.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatMicroTime(t *metav1.MicroTime) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

func sortedKeys[K ~string, V any](m map[K]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return keys
}
