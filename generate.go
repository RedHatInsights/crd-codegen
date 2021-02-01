package generate

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"encoding/json"

	typegen "github.com/a-h/generate"
	goyaml "gopkg.in/yaml.v2"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	k8syaml "sigs.k8s.io/yaml"
)

var sch = runtime.NewScheme()

func init() {
	metav1.AddToGroupVersion(sch, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(scheme.AddToScheme(sch))
	utilruntime.Must(apiextv1beta1.AddToScheme(sch))
}

// DecodeCRD marshals a JSON representation of a CRD into a CustomResourceDefition
func DecodeCRD(jsonBytes []byte) (*apiextv1beta1.CustomResourceDefinition, error) {
	// decode as an unstructured object and then cast to CRD, since I
	// was having issues decoding straight to the CRD obj.
	// See https://github.com/openshift/origin/pull/21936/files
	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, jsonBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load JSON into unstructured object: %v", err)
	}

	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unable to cast %T to *unstructured.Unstructured", obj)
	}

	gvk := unstructuredObj.GroupVersionKind()
	newObj, err := sch.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("unable to create new scheme: %v", err)
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, newObj); err != nil {
		return nil, fmt.Errorf("unable to convert unstructured object to '%v': %v", gvk, err)
	}

	crd, ok := newObj.(*apiextv1beta1.CustomResourceDefinition)
	if !ok {
		return nil, fmt.Errorf("unable to cast %T to *apiextv1beta1.CustomResourceDefinition", obj)
	}

	return crd, nil
}

// SplitYaml splits a multi-doc YAML file into a list of files
func SplitYaml(resources []byte) ([][]byte, error) {
	// https://gist.github.com/yanniszark/c6f347421a1eeb75057ff421e03fd57c
	dec := goyaml.NewDecoder(bytes.NewReader(resources))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

// ExtractSchema pulls out the openApiV3Schema from a CRD
func ExtractSchema(jsonDoc []byte) (*schema.GroupVersionKind, []byte, error) {
	if string(jsonDoc) == "null" {
		// doc is empty
		return nil, nil, nil
	}

	crd, err := DecodeCRD(jsonDoc)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing crd: %v", err)
	}

	gvk := &schema.GroupVersionKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}
	if crd.Spec.Version != "" {
		gvk.Version = crd.Spec.Version
	} else {
		gvk.Version = crd.Spec.Versions[0].Name
	}

	log.Printf("extracting schema for %s/%s %s", gvk.Group, gvk.Version, gvk.Kind)

	schemaJSON := *crd.Spec.Validation.OpenAPIV3Schema
	schemaJSON.Schema = "http://json-schema.org/draft-07/schema#"
	schemaJSON.Title = gvk.Kind

	output, err := json.MarshalIndent(schemaJSON, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal json: %v", err)
	}

	return gvk, output, nil

}

// generateTypes is adapted from json-schema-generate's main.go
func generateTypes(schemaFilePath string, pkgName string, outPath string) error {
	schemas, err := typegen.ReadInputFiles([]string{schemaFilePath}, false)
	if err != nil {
		return fmt.Errorf("failure reading schema file: %v", err)
	}

	g := typegen.New(schemas...)

	err = g.CreateTypes()
	if err != nil {
		return fmt.Errorf("failure generating structs: %v", err)
	}

	w, err := os.Create(outPath)

	if err != nil {
		return fmt.Errorf("error opening output file: %v", err)
	}

	typegen.Output(w, g, pkgName) //false, false)

	return nil
}

// Generate parses an input CRD YAML file and outputs JSON schemas/Go types to an output dir
func Generate(in string, to string) error {
	// read yaml
	yamlFile, err := ioutil.ReadFile(in)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	// split multidoc YAML and convert to JSON
	yamlDocs, err := SplitYaml(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to parse YAML file: %v", err)
	}

	for _, yamlDoc := range yamlDocs {
		jsonDoc, err := k8syaml.YAMLToJSON(yamlDoc)
		if err != nil {
			return fmt.Errorf("failed to convert to json: %v", err)
		}
		gvk, schemaJSON, err := ExtractSchema(jsonDoc)
		if err != nil {
			return fmt.Errorf("failed to extract schema: %v", err)
		}

		if schemaJSON != nil {
			// write JSON schemas to {outputDir}/schemas
			schemaFileDir := filepath.Join(to, "schemas")
			if err := os.MkdirAll(schemaFileDir, 0755); err != nil {
				return fmt.Errorf("unable to create output directory: %v", err)
			}
			schemaFilePath := filepath.Join(schemaFileDir, fmt.Sprintf("%s-%s-%s-schema.json", gvk.Group, gvk.Kind, gvk.Version))
			err = ioutil.WriteFile(schemaFilePath, schemaJSON, 0644)
			if err != nil {
				return fmt.Errorf("failed to write schema file '%s': %v", schemaFilePath, err)
			}

			// generate types to {outputDir}/apis/{group}/{version}/{kind}_types.go
			typeFileDir := filepath.Join(to, "apis", gvk.Group, gvk.Version)
			if err := os.MkdirAll(typeFileDir, 0755); err != nil {
				return fmt.Errorf("unable to create output directory: %v", err)
			}
			typeFilePath := filepath.Join(typeFileDir, fmt.Sprintf("%s_types.go", strings.ToLower(gvk.Kind)))
			err = generateTypes(schemaFilePath, gvk.Version, typeFilePath)
			if err != nil {
				return fmt.Errorf("failed to generate types: %v", err)
			}
		}
	}

	return nil
}
