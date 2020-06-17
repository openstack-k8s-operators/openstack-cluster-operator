# NOTE: requires installation of 'opm': see https://github.com/operator-framework/operator-registry/releases
INDEX_VERSION=${INDEX_VERSION:-"0.0.1"}
BUNDLE_VERSION=${BUNDLE_VERSION:-"0.0.1"}
IMAGE_REGISTRY=${IMAGE_REGISTRY:?"Please define a IMAGE_REGISTRY. Example: quay.io/username"}
#opm index add --bundles $IMAGE_REGISTRY/openstack-cluster-bundle:v$BUNDLE_VERSION --tag $IMAGE_REGISTRY/openstack-operators-index:v$INDEX_VERSION
opm index add --bundles $BUNDLE_IMAGE --tag $IMAGE_REGISTRY/openstack-operators-index:v$INDEX_VERSION
podman push $IMAGE_REGISTRY/openstack-operators-index:v$INDEX_VERSION
