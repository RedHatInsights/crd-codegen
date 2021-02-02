# crd-codegen

Generates Go API interfaces for a Kubernetes Custom Resource by parsing its OpenAPI schema in the YAML `CustomResourceDefinition`. Useful if an operator/application was not written in Go but you have a need to interact with its resoures.

This is a **work in progress**. The JSON schema generators do not generate code that is fully-compatible with k8s controller-gen, however getting more of the process automated is in the works.

## Example usage

Using the Strimzi operator as an example:

```
# grab dependencies
go get
export PATH=$PATH:$(go env GOPATH)

# download CRD's for strimzi operator
VERSION=0.21.1
wget https://github.com/strimzi/strimzi-kafka-operator/releases/download/${VERSION}/strimzi-crds-${VERSION}.yaml

# generate types using the CRD
./generate.sh strimzi-crds-${VERSION}.yaml

# generated apis are found in 'generated' directory:
ls generated/apis/kafka.strimzi.io/v1beta1/

# KafkaConnect.go  KafkaConnectS2I.go  Kafka.go  KafkaMirrorMaker.go  KafkaTopic.go  KafkaUser.go
```
