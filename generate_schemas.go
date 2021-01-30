package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"encoding/json"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

// JSONSchema describes json schema file
type JSONSchema struct {
	Schema      string                                   `json:"$schema"`
	Ref         string                                   `json:"$ref"`
	ID          string                                   `json:"$id"`
	Title       string                                   `json:"$title"`
	Definitions map[string]apiextv1beta1.JSONSchemaProps `json:"definitions"`
}

var sch = runtime.NewScheme()

func init() {
	metav1.AddToGroupVersion(sch, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(scheme.AddToScheme(sch))
	utilruntime.Must(apiextv1beta1.AddToScheme(sch))
}

func decodeCRD(jsonBytes []byte) (*apiextv1beta1.CustomResourceDefinition, error) {
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

func generate(in *string, out *string) {
	// read yaml
	yb, err := ioutil.ReadFile(*in)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	// convert to json
	jb, err := yaml.YAMLToJSON(yb)
	if err != nil {
		log.Fatalf("failed to convert to json: %v", err)
	}
	crd, err := decodeCRD(jb)
	if err != nil {
		log.Fatalf("error parsing crd: %v", err)
	}

	kind := crd.Spec.Names.Kind

	schema := &JSONSchema{
		Schema: "http://json-schema.org/draft-07/schema#",
		Ref:    fmt.Sprintf("#/definitions/%s", kind),
		ID:     kind,
		Title:  kind,
		//Definitions: make(map[string]JSONSchemaDefinition),
		Definitions: make(map[string]apiextv1beta1.JSONSchemaProps),
	}
	crd.Spec.Validation.OpenAPIV3Schema.Type = "object"
	crd.Spec.Validation.OpenAPIV3Schema.Description = kind
	schema.Definitions[kind] = *crd.Spec.Validation.OpenAPIV3Schema

	output, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal json: %v", err)
	}
	err = ioutil.WriteFile(*out, output, 0644)
	if err != nil {
		log.Fatalf("failed to write file: %v", err)
	}
}

func main() {
	in := flag.String("in", "", "path to strimzi CRD yaml file")
	out := flag.String("out", "", "path to output json file")
	flag.Parse()

	if *in == "" {
		log.Fatalf("input file must be specified with -in=")
	}

	if *out == "" {
		log.Fatalf("output file must be specified with -out=")
	}

	generate(in, out)
}
