package services

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	yamlutil "sigs.k8s.io/yaml"
)

func parseManifest(data []byte) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		// Skip comment-only documents
		hasContent := false
		for _, line := range strings.Split(string(doc), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && trimmed != "---" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			continue
		}

		jsonData, err := yamlutil.YAMLToJSON(doc)
		if err != nil {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := obj.UnmarshalJSON(jsonData); err != nil {
			continue
		}

		if obj.GetKind() == "" {
			continue
		}

		objects = append(objects, obj)
	}

	return objects, nil
}
