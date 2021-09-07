#!/bin/bash
set -e

go build generate_schemas.go

if [ "$#" -eq "0" ]; then
    echo "usage: generate.sh <CRD yaml file>"
    exit 1
fi

if ! test -f $1; then
    echo "file not found: $1"
    exit 1
fi

rm -fr generated
mkdir -p generated
mkdir -p generated/crds
mkdir -p generated/schemas
mkdir -p generated/apis

# Split the multi-doc YAML into separate files since we can't unmarshal multidoc
NAMES=$(yq eval --no-doc 'select(.metadata.name) | .metadata.name' $1 | xargs)
for name in $NAMES; do
    yq eval "select(.metadata.name == \"$name\")" $1 > generated/crds/$name.yaml
    ./generate_schemas -in=generated/crds/$name.yaml -out=generated/schemas/$name-schema.json
    CR_GROUP=$(yq eval ".spec.group" generated/crds/$name.yaml)
    CR_KIND=$(yq eval ".spec.names.kind" generated/crds/$name.yaml)
    CR_VER=$(yq eval ".spec.versions[0].name" generated/crds/$name.yaml)
    gojsonschema -p $CR_VER -o generated/apis/$CR_GROUP/$CR_VER/$CR_KIND.go generated/schemas/$name-schema.json
done
