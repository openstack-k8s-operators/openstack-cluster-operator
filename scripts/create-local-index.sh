# Creates local bundle and index images and pushes them into the local openshift registry.

# NOTE: requires installation of 'opm': see https://github.com/operator-framework/operator-registry/releases
# NOTE: this script assumes you have configured your OpenShift registry with a default route so that it is easy to push images locally
set -ex
BUNDLE_IMAGE=${BUNDLE_IMAGE:?"Please define a bundle image."}
BUNDLE_VERSION=${BUNDLE_VERSION:-"0.0.1"}
INDEX_VERSION=${INDEX_VERSION:-"0.0.1"}
PROJECT=${PROJECT:-"openstack"}
IMAGE_REGISTRY=$(oc get route -n openshift-image-registry -o json | jq ".items[0].spec.host" -r)

if ! oc project | grep "$PROJECT" &> /dev/null; then echo "run this script in the project: $PROJECT"; exit 1; fi

ACCOUNT=$(oc get secret | grep builder-dockercfg | cut -f 1 -d ' ')
TOKEN=$(oc get secret $ACCOUNT -o json | jq '.metadata.annotations["openshift.io/token-secret.value"]' -r)
echo "$TOKEN" | podman login -u $ACCOUNT --password-stdin --tls-verify=false $IMAGE_REGISTRY

#FIXME: need a better way to reconcile the internal and external image names from openshift
# the opm command below when executed outside the cluster requires access to the bundle image on the CLI
# but when deployed the database needs to have the internal location recorded
opm index add --generate --bundles $BUNDLE_IMAGE --tag $IMAGE_REGISTRY/$PROJECT/openstack-operators-index:v$INDEX_VERSION -c podman
sqlite3 database/index.db "UPDATE operatorbundle SET bundlepath=\"image-registry.openshift-image-registry.svc:5000/$PROJECT/openstack-cluster-bundle:v$BUNDLE_VERSION\"" ".exit"
buildah bud -t $IMAGE_REGISTRY/$PROJECT/openstack-operators-index:v$INDEX_VERSION index.Dockerfile
podman push --tls-verify=false $IMAGE_REGISTRY/$PROJECT/openstack-operators-index:v$INDEX_VERSION
