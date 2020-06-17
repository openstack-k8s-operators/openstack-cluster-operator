# Creates local bundle and index images and pushes them into the local openshift registry.

# NOTE: requires installation of 'opm': see https://github.com/operator-framework/operator-registry/releases
# NOTE: this script assumes you have configured your OpenShift registry with a default route so that it is easy to push images locally
set -ex
BUNDLE_IMAGE=${BUNDLE_IMAGE:?"Please define a bundle image."}
INDEX_VERSION=${INDEX_VERSION:-"0.0.1"}
PROJECT=${PROJECT:-"openstack"}
IMAGE_REGISTRY=$(oc get route -n openshift-image-registry -o json | jq ".items[0].spec.host" -r)

if ! oc project | grep "$PROJECT" &> /dev/null; then echo "run this script in the project: $PROJECT"; exit 1; fi

ACCOUNT=$(oc get secret | grep builder-dockercfg | cut -f 1 -d ' ')
TOKEN=$(oc get secret $ACCOUNT -o json | jq '.metadata.annotations["openshift.io/token-secret.value"]' -r)
echo "$TOKEN" | podman login -u $ACCOUNT --password-stdin --tls-verify=false $IMAGE_REGISTRY

opm index add --bundles $BUNDLE_IMAGE --tag $IMAGE_REGISTRY/$PROJECT/openstack-operators-index:v$INDEX_VERSION --permissive

podman push --tls-verify=false $IMAGE_REGISTRY/$PROJECT/openstack-operators-index:v$INDEX_VERSION
