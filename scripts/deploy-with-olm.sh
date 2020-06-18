set -x

INDEX_IMAGE=${INDEX_IMAGE:-"quay.io/openstack-k8s-operators/openstack-operators-index:v0.0.1"}
CSV_VERSION=${CSV_VERSION:-"0.0.1"}

if [ `oc get catalogsource -n "${TARGET_NAMESPACE}" --no-headers 2> /dev/null | grep openstack | wc -l` -eq 0 ]; then
echo "Creating CatalogSource"
        cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: openstack-index
  namespace: openstack
spec:
  sourceType: grpc
  image: $INDEX_IMAGE
EOF
fi

echo "Wait for the catalogSource to be available"
oc wait deploy "openstack-index" --for condition=available -n openstack --timeout="360s"

RETRIES="${RETRIES:-10}"
for i in $(seq 1 $RETRIES); do
    echo "Waiting for packagemanifest 'openstack-cluster' to be created in namespace 'openstack'..."
    oc get packagemanifest -n "openstack" "openstack-cluster" && break
    sleep $i
    if [ "$i" -eq "${RETRIES}" ]; then
      echo "packagemanifest 'openstack-cluster' was never created in namespace 'openstack'"
      exit 1
    fi
done

if [ `oc get operatorgroup -n "${TARGET_NAMESPACE}" --no-headers 2> /dev/null | wc -l` -eq 0 ]; then
echo "Creating OperatorGroup"
cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: "openstack-group"
  namespace: "openstack"
spec:
  targetNamespaces:
  - "openstack"
EOF
fi

echo "Creating Subscription"
cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openstack-cluster
  namespace: openstack
spec:
  source: openstack-index
  sourceNamespace: openstack
  name: openstack-cluster
  startingCSV: openstack-cluster-operator.v$CSV_VERSION
  channel: beta
  installPlanApproval: manual
EOF
