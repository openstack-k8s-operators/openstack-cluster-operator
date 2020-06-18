# Creates local bundle and index images and pushes them into the local openshift registry.

# NOTE: requires installation of 'opm': see https://github.com/operator-framework/operator-registry/releases
# NOTE: this script assumes you have configured your OpenShift registry with a default route so that it is easy to push images locally
set -ex
BUNDLE_VERSION=${BUNDLE_VERSION:-"0.0.1"}
PROJECT=${PROJECT:-"openstack"}
IMAGE_REGISTRY=$(oc get route -n openshift-image-registry -o json | jq ".items[0].spec.host" -r)

if ! oc project | grep "$PROJECT" &> /dev/null; then echo "run this script in the project: $PROJECT"; exit 1; fi

ACCOUNT=$(oc get secret | grep builder-dockercfg | cut -f 1 -d ' ')
TOKEN=$(oc get secret $ACCOUNT -o json | jq '.metadata.annotations["openshift.io/token-secret.value"]' -r)
echo "$TOKEN" | podman login -u $ACCOUNT --password-stdin --tls-verify=false $IMAGE_REGISTRY

opm alpha bundle build -d deploy/olm-catalog/openstack-cluster/$BUNDLE_VERSION/ --package openstack-cluster --channels beta --default beta --tag $IMAGE_REGISTRY/openstack-cluster-bundle:v$BUNDLE_VERSION -b buildah
podman push $IMAGE_REGISTRY/$PROJECT/openstack-cluster-bundle:v$BUNDLE_VERSION
opm alpha bundle validate --tag $IMAGE_REGISTRY/$PROJECT/openstack-cluster-bundle:v$BUNDLE_VERSION -b podman
rm bundle.Dockerfile
