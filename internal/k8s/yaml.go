package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
)

// cleanupYAMLObject удаляет служебные поля из объекта
func cleanupYAMLObject(obj interface{}) interface{} {
	// Используем JSON для быстрого преобразования в map (JSON->struct быстрее чем YAML)
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return obj
	}

	var cleaned interface{}
	err = json.Unmarshal(jsonBytes, &cleaned)
	if err != nil {
		return obj
	}

	// Удаляем managedFields и selfLink рекурсивно
	cleaned = removeFields(cleaned)
	return cleaned
}

// removeFields рекурсивно удаляет служебные поля (только managedFields)
func removeFields(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for k, val := range v {
			if k == "managedFields" {
				continue
			}
			result[k] = removeFields(val)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, val := range v {
			if k == "managedFields" {
				continue
			}
			result[k] = removeFields(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = removeFields(val)
		}
		return result
	default:
		return v
	}
}

// getKindName возвращает правильное имя Kind для типа
func getKindName(kind string) string {
	// Преобразуем в правильный формат
	kindMap := map[string]string{
		"pod":                "Pod",
		"deployment":         "Deployment",
		"service":            "Service",
		"configmap":          "ConfigMap",
		"secret":             "Secret",
		"node":               "Node",
		"serviceaccount":     "ServiceAccount",
		"role":               "Role",
		"clusterrole":        "ClusterRole",
		"rolebinding":        "RoleBinding",
		"clusterrolebinding": "ClusterRoleBinding",
	}
	if k, ok := kindMap[kind]; ok {
		return k
	}
	return kind
}

// reorderMetadata переупорядочивает поля metadata в нужном порядке
func reorderMetadata(metadata map[string]interface{}) map[string]interface{} {
	ordered := make(map[string]interface{})

	// Правильный порядок полей metadata
	fields := []string{"name", "generateName", "namespace", "uid", "resourceVersion", "creationTimestamp", "labels", "annotations", "ownerReferences", "selfLink"}

	for _, field := range fields {
		if val, ok := metadata[field]; ok {
			ordered[field] = val
		}
	}

	// Добавляем остальные поля которых нет в списке
	for k, v := range metadata {
		if _, exists := ordered[k]; !exists {
			ordered[k] = v
		}
	}

	return ordered
}

// reorderTopLevel переупорядочивает поля верхнего уровня
func reorderTopLevel(data map[string]interface{}) map[string]interface{} {
	ordered := make(map[string]interface{})

	// Порядок: apiVersion, kind, metadata, spec, status
	fields := []string{"apiVersion", "kind", "metadata", "spec", "status"}

	for _, field := range fields {
		if val, ok := data[field]; ok {
			ordered[field] = val
		}
	}

	// Добавляем остальные поля
	for k, v := range data {
		if _, exists := ordered[k]; !exists {
			ordered[k] = v
		}
	}

	return ordered
}

// cleanYAMLForEditing удаляет служебные поля которые генерирует Kubernetes
func cleanYAMLForEditing(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Копируем только нужные поля
	for k, v := range data {
		// Пропускаем status - это служебное поле
		if k == "status" {
			continue
		}
		// Пропускаем managedFields - служебное поле
		if k == "managedFields" {
			continue
		}
		result[k] = v
	}

	// Очищаем metadata от служебных полей
	if metadata, ok := result["metadata"].(map[string]interface{}); ok {
		cleanedMetadata := make(map[string]interface{})
		for k, v := range metadata {
			// Удаляем служебные поля из metadata
			switch k {
			case "resourceVersion", "uid", "selfLink", "managedFields":
				continue
			default:
				cleanedMetadata[k] = v
			}
		}
		result["metadata"] = cleanedMetadata
	}

	// Очищаем spec от служебных volume mount'ов для сервис аккаунта
	if spec, ok := result["spec"].(map[string]interface{}); ok {
		if containers, ok := spec["containers"].([]interface{}); ok {
			cleanedContainers := make([]interface{}, 0)
			for _, container := range containers {
				if containerMap, ok := container.(map[string]interface{}); ok {
					if volumeMounts, ok := containerMap["volumeMounts"].([]interface{}); ok {
						cleanedVolumeMounts := make([]interface{}, 0)
						for _, volumeMount := range volumeMounts {
							if vmMap, ok := volumeMount.(map[string]interface{}); ok {
								// Пропускаем автоматически добавляемый serviceaccount volume mount
								if mountPath, ok := vmMap["mountPath"].(string); ok {
									if mountPath == "/var/run/secrets/kubernetes.io/serviceaccount" {
										continue
									}
								}
							}
							cleanedVolumeMounts = append(cleanedVolumeMounts, volumeMount)
						}
						if len(cleanedVolumeMounts) > 0 {
							containerMap["volumeMounts"] = cleanedVolumeMounts
						} else {
							delete(containerMap, "volumeMounts")
						}
					}
				}
				cleanedContainers = append(cleanedContainers, container)
			}
			spec["containers"] = cleanedContainers
		}

		// Удаляем автоматически добавляемый volume для serviceaccount
		if volumes, ok := spec["volumes"].([]interface{}); ok {
			cleanedVolumes := make([]interface{}, 0)
			for _, volume := range volumes {
				if volumeMap, ok := volume.(map[string]interface{}); ok {
					if name, ok := volumeMap["name"].(string); ok {
						// Пропускаем автоматический volume для serviceaccount (начинается с kube-api-access)
						if len(name) > len("kube-api-access") && name[:len("kube-api-access")] == "kube-api-access" {
							continue
						}
					}
				}
				cleanedVolumes = append(cleanedVolumes, volume)
			}
			if len(cleanedVolumes) > 0 {
				spec["volumes"] = cleanedVolumes
			} else {
				delete(spec, "volumes")
			}
		}
		result["spec"] = spec
	}

	return result
}

func (m *Manager) GetResourceYAML(namespace, kind, name string, clean bool) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var data interface{}

	switch kind {
	case "pod":
		data, err = conn.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		data, err = conn.Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "service":
		data, err = conn.Client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	case "configmap":
		data, err = conn.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	case "secret":
		data, err = conn.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "node":
		data, err = conn.Client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	case "serviceaccount":
		data, err = conn.Client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	case "role":
		data, err = conn.Client.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	case "rolebinding":
		data, err = conn.Client.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	case "clusterrolebinding":
		data, err = conn.Client.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	case "clusterrole":
		dynClient, dynErr := dynamic.NewForConfig(conn.Config)
		if dynErr != nil {
			err = dynErr
			break
		}
		gvr := schema.GroupVersionResource{
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		}
		unstructured, dynErr := dynClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
		if dynErr != nil {
			err = dynErr
			break
		}
		data = unstructured.UnstructuredContent()
	default:
		// Для неизвестных ресурсов используем dynamic client
		return m.getDynamicYAML(conn, ctx, namespace, kind, name, clean)
	}

	fmt.Println("GetResourceYAML: kind=", kind, " name=", name, " namespace=", namespace, " data=", data, " err=", err)

	if err != nil {
		return "", fmt.Errorf("не удалось получить %s/%s: %w", kind, name, err)
	}

	if data == nil {
		return "", fmt.Errorf("данные для %s/%s пусты", kind, name)
	}

	cleaned := cleanupYAMLObject(data)
	if cleaned == nil {
		return "", fmt.Errorf("не удалось очистить данные %s/%s", kind, name)
	}

	cleanedMap, ok := cleaned.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("неверный тип данных для %s/%s", kind, name)
	}

	fmt.Println("cleanedMap: ", cleanedMap)

	// Очищаем от служебных полей если нужно
	if clean {
		cleanedMap = cleanYAMLForEditing(cleanedMap)
	}

	fmt.Println("cleanedMap after clean: ", cleanedMap)

	// Переупорядочиваем metadata если она есть
	if metadata, ok := cleanedMap["metadata"].(map[string]interface{}); ok {
		cleanedMap["metadata"] = reorderMetadata(metadata)
	}

	// Добавляем apiVersion и kind вручную
	ordered := make(map[string]interface{})

	// Правильная apiVersion для разных типов ресурсов
	switch kind {
	case "role", "clusterrole", "rolebinding", "clusterrolebinding":
		ordered["apiVersion"] = "rbac.authorization.k8s.io/v1"
	default:
		ordered["apiVersion"] = "v1"
	}

	ordered["kind"] = getKindName(kind)

	// Копируем остальные поля
	for k, v := range cleanedMap {
		ordered[k] = v
	}

	// Переупорядочиваем поля верхнего уровня
	ordered = reorderTopLevel(ordered)

	fmt.Println("ordered: ", ordered)

	yamlBytes, err := yaml.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}

	fmt.Println("yamlBytes: ", string(yamlBytes))

	return string(yamlBytes), nil
}

func (m *Manager) getDynamicYAML(conn *ClusterConnection, ctx context.Context, namespace, kind, name string, clean bool) (string, error) {
	dynClient, err := dynamic.NewForConfig(conn.Config)
	if err != nil {
		return "", err
	}

	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: kind + "s",
	}

	var obj interface{}
	if namespace != "" {
		obj, err = dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = dynClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return "", err
	}

	obj = cleanupYAMLObject(obj)
	cleanedMap := obj.(map[string]interface{})

	// Очищаем от служебных полей если нужно
	if clean {
		cleanedMap = cleanYAMLForEditing(cleanedMap)
	}

	// Переупорядочиваем metadata если она есть
	if metadata, ok := cleanedMap["metadata"].(map[string]interface{}); ok {
		cleanedMap["metadata"] = reorderMetadata(metadata)
	}

	// Добавляем apiVersion и kind вручную
	ordered := make(map[string]interface{})

	// Правильная apiVersion для разных типов ресурсов
	switch kind {
	case "role", "clusterrole", "rolebinding", "clusterrolebinding":
		ordered["apiVersion"] = "rbac.authorization.k8s.io/v1"
	default:
		ordered["apiVersion"] = "v1"
	}

	ordered["kind"] = getKindName(kind)

	// Копируем остальные поля
	for k, v := range cleanedMap {
		ordered[k] = v
	}

	// Переупорядочиваем поля верхнего уровня
	ordered = reorderTopLevel(ordered)

	yamlBytes, err := yaml.Marshal(ordered)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}
