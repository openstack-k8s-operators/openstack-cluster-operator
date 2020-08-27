#!/usr/bin/env bash
set -ex

# build-manifests is designed to populate the deploy directory
# with all of the manifests necessary for use in development
# and for consumption with the operator-lifecycle-manager.

# individual bash functions below are implemented for each
# component operator that generate individual CSV's via the
# csv-generate utilities deployed along with each operator.
# Versions can be passed into these utilities to control the
# container images that get used.

# These resulting individual component CSV's are then merged
# into an "openstack-cluster" using the csv-merge tool in this
# project. This generates the final (completed) CSV containing
# all the needed settings to install each operator.

# CRD's are then copied out of the container for each component
# operator into the deploy directory. Once this as been done
# an Operator bundle can be created.

PROJECT_ROOT="$(readlink -e $(dirname "$BASH_SOURCE[0]")/../)"
#source "${PROJECT_ROOT}"/script/config

# REPLACES_VERSION is the old CSV_VERSION
#   if REPLACES_VERSION == "" it will be ignored
REPLACES_CSV_VERSION="${REPLACES_VERSION:-}"
CSV_VERSION="${CSV_VERSION:-0.0.1}" #this should match the BUNDLE_VERSION you set if using scripts/create-bundle.sh
CONTAINER_BUILD_CMD="${CONTAINER_BUILD_CMD:-podman}"

DEPLOY_DIR="${PROJECT_ROOT}/deploy"
#CRD_DIR="${DEPLOY_DIR}/crds"
CSV_DIR="${DEPLOY_DIR}/olm-catalog/openstack-cluster/${CSV_VERSION}"

OPERATOR_NAME="${NAME:-openstack-cluster-operator}"
OPERATOR_NAMESPACE="${NAMESPACE:-openstack-cluster-operator}"
OPERATOR_IMAGE="${OPERATOR_IMAGE:-quay.io/openstack-k8s-operators/openstack-cluster-operator:v0.0.1}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

# Component Images
NOVA_IMAGE="${NOVA_IMAGE:-quay.io/openstack-k8s-operators/nova-operator:v0.0.3}"
NEUTRON_IMAGE="${NEUTRON_IMAGE:-quay.io/openstack-k8s-operators/neutron-operator:v0.0.3}"
COMPUTE_WORKER_IMAGE="${COMPUTE_WORKER_IMAGE:-quay.io/openstack-k8s-operators/compute-node-operator:v0.0.3}"
KEYSTONE_IMAGE="${KEYSTONE_IMAGE:-quay.io/openstack-k8s-operators/keystone-operator:v0.0.2}"
HEAT_IMAGE="${HEAT_IMAGE:-quay.io/openstack-k8s-operators/heat-operator:devel}"
GLANCE_IMAGE="${GLANCE_IMAGE:-quay.io/openstack-k8s-operators/glance-operator:devel}"
MARIADB_IMAGE="${MARIADB_IMAGE:-quay.io/openstack-k8s-operators/mariadb-operator:v0.0.1}"

# Important extensions
CSV_EXT="clusterserviceversion.yaml"
CSV_CRD_EXT="csv_crds.yaml"

function copy_deployment_specs() {

  local operatorName="$1" && shift
  local imagePullUrl="$1" && shift
  local copyToDir="$1" && shift

  mkdir -p "$copyToDir/$operatorName"

  ID=$($CONTAINER_BUILD_CMD run -d --entrypoint="/bin/sleep" $imagePullUrl 10)
  # NOTE: This requires as a convention we will copy deployment specs and CRDs to for example usr/share/nova-operator/bundle/ in our operator images
  $CONTAINER_BUILD_CMD cp "$ID:/usr/share/$operatorName/bundle/" "$copyToDir/$operatorName"
  $CONTAINER_BUILD_CMD kill "$ID"
  $CONTAINER_BUILD_CMD rm -f "$ID"

  for FILE in $(ls "$copyToDir/$operatorName/bundle"); do
      mv "$copyToDir/$operatorName/bundle/$FILE" ${copyToDir}/${operatorName}-$FILE
  done
  rm -Rf "$copyToDir/$operatorName"

}

function gen_csv() {
  # Handle arguments
  local operatorName="$1" && shift
  local imagePullUrl="$1" && shift
  local dumpCRDsArg="$1" && shift
  local operatorArgs="$@"

  # Handle important vars
  local csv="${operatorName}.${CSV_EXT}"
  #local csvWithCRDs="${operatorName}.${CSV_CRD_EXT}"
  local crds="${operatorName}.crds.yaml"

  # TODO: Use oc to run if cluster is available
  local containerBuildCmd="$CONTAINER_BUILD_CMD run --rm --entrypoint=/usr/local/bin/csv-generator ${imagePullUrl} ${operatorArgs}"

  eval $containerBuildCmd > $csv
}

function create_nova_csv() {
  local operatorName="nova"
  local imagePullUrl="${NOVA_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${NOVA_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_neutron_csv() {
  local operatorName="neutron"
  local imagePullUrl="${NEUTRON_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${NEUTRON_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_compute_node_csv() {
  local operatorName="compute-node"
  local imagePullUrl="${COMPUTE_WORKER_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${COMPUTE_WORKER_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_keystone_csv() {
  local operatorName="keystone"
  local imagePullUrl="${KEYSTONE_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${KEYSTONE_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_heat_csv() {
  local operatorName="heat"
  local imagePullUrl="${HEAT_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${HEAT_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_glance_csv() {
  local operatorName="glance"
  local imagePullUrl="${GLANCE_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${GLANCE_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

function create_mariadb_csv() {
  local operatorName="mariad"
  local imagePullUrl="${MARIADB_IMAGE}"
  local operatorArgs=" \
    --namespace=${OPERATOR_NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --operator-image-name=${MARIADB_IMAGE}
  "

  gen_csv ${operatorName} ${imagePullUrl} ${operatorArgs}
  echo "${operatorName}"
}

TEMPDIR=$(mktemp -d) || (echo "Failed to create temp directory" && exit 1)
pushd $TEMPDIR
novaCsv="${TEMPDIR}/$(create_nova_csv).${CSV_EXT}"
neutronCsv="${TEMPDIR}/$(create_neutron_csv).${CSV_EXT}"
computeNodeCsv="${TEMPDIR}/$(create_compute_node_csv).${CSV_EXT}"
keystoneCsv="${TEMPDIR}/$(create_keystone_csv).${CSV_EXT}"
heatCsv="${TEMPDIR}/$(create_heat_csv).${CSV_EXT}"
mariadbCsv="${TEMPDIR}/$(create_mariadb_csv).${CSV_EXT}"
glanceCsv="${TEMPDIR}/$(create_glance_csv).${CSV_EXT}"
csvOverrides="${TEMPDIR}/csv_overrides.${CSV_EXT}"
cat > ${csvOverrides} <<- EOM
---
spec:
  links:
  - name: OpenStack K8s
    url: https://github.com/openstack-k8s-operators
  - name: Source Code
    url: https://github.com/openstack-k8s-operators/
  maintainers:
  - email: openstack-discuss@lists.openstack.org
    name: OpenStack K8s
  maturity: alpha
  provider:
    name: OpenStack K8s
EOM

rm -Rf "${CSV_DIR}"
mkdir -p "${CSV_DIR}"

  #--spec-description="$(<${PROJECT_ROOT}/docs/operator_description.md)" \
# Build and merge CSVs
${PROJECT_ROOT}/bin/csv-merger \
  --nova-csv="$(<${novaCsv})" \
  --neutron-csv="$(<${neutronCsv})" \
  --compute-node-csv="$(<${computeNodeCsv})" \
  --keystone-csv="$(<${keystoneCsv})" \
  --glance-csv="$(<${glanceCsv})" \
  --mariadb-csv="$(<${mariadbCsv})" \
  --csv-version=${CSV_VERSION} \
  --replaces-csv-version=${REPLACES_CSV_VERSION} \
  --spec-displayname="OpenStack Cluster Operator" \
  --spec-description="Installs OpenStack Cluster Operator" \
  --crd-display="OpenStack Cluster Operator" \
  -csv-overrides="$(<${csvOverrides})" \
  --operator-image-name="${OPERATOR_IMAGE}" > "${CSV_DIR}/${OPERATOR_NAME}.v${CSV_VERSION}.${CSV_EXT}"
(cd ${PROJECT_ROOT}/tools/csv-merger/ && go clean)

copy_deployment_specs "openstack-cluster-operator" "${OPERATOR_IMAGE}" "$CSV_DIR"
copy_deployment_specs "nova-operator" "${NOVA_IMAGE}" "$CSV_DIR"
copy_deployment_specs "neutron-operator" "${NEUTRON_IMAGE}" "$CSV_DIR"
copy_deployment_specs "compute-node-operator" "${COMPUTE_WORKER_IMAGE}" "$CSV_DIR"
copy_deployment_specs "keystone-operator" "${KEYSTONE_IMAGE}" "$CSV_DIR"
copy_deployment_specs "mariadb-operator" "${MARIADB_IMAGE}" "$CSV_DIR"
copy_deployment_specs "glance-operator" "${GLANCE_IMAGE}" "$CSV_DIR"

rm -rf ${TEMPDIR}
