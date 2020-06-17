# NOTE: requires installation of 'opm': see https://github.com/operator-framework/operator-registry/releases
set -x
BUNDLE_VERSION=${BUNDLE_VERSION:-"0.0.1"}
IMAGE_REGISTRY=${IMAGE_REGISTRY:?"Please define a IMAGE_REGISTRY. Example: quay.io/username"}
opm alpha bundle build -d deploy/olm-catalog/openstack-cluster/$BUNDLE_VERSION/ --package openstack-cluster --channels beta --default beta --tag $IMAGE_REGISTRY/openstack-cluster-bundle:v$BUNDLE_VERSION -b buildah
podman push $IMAGE_REGISTRY/openstack-cluster-bundle:v$BUNDLE_VERSION
opm alpha bundle validate --tag $IMAGE_REGISTRY/openstack-cluster-bundle:v$BUNDLE_VERSION -b podman
rm bundle.Dockerfile
