# crd-codegen

Generates Go type definitions for a Kubernetes Custom Resource by parsing a YAML `CustomResourceDefinition`. Useful if an operator/application was not written in Go but you have a need to interact with its resoures.

This tool extracts the OpenAPIv3 spec from the CRD and runs it through a JSON schema type generator.

## Example usage

Using the Strimzi operator as an example:

```
# grab dependencies
go get -u github.com/bsquizz/crd-codegen/...
export PATH=$PATH:$(go env GOPATH)

# download CRD's for strimzi operator
VERSION=0.21.1
wget https://github.com/strimzi/strimzi-kafka-operator/releases/download/${VERSION}/strimzi-crds-${VERSION}.yaml

# generate types using the CRD
crd-codegen -in=strimzi-crds-${VERSION}.yaml -to=generated

ls generated/apis/kafka.strimzi.io/v1beta1/
# kafkaconnects2i_types.go  kafkaconnect_types.go  kafkamirrormaker_types.go  kafkatopic_types.go  kafka_types.go  kafkauser_types.go
```
