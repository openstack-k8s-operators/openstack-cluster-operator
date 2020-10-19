/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	uns "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controlplanev1beta1 "github.com/openstack-k8s-operators/openstack-cluster-operator/api/v1beta1"
	bindatautil "github.com/openstack-k8s-operators/openstack-cluster-operator/pkg/bindata_util"
)

// ManifestPath - bindata path
var ManifestPath = "./bindata"

const (
	ownerUIDLabelSelector       = "controlplane.openstack.org/uid"
	ownerNameSpaceLabelSelector = "controlplane.openstack.org/namespace"
	ownerNameLabelSelector      = "controlplane.openstack.org/name"
)

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=controlplane.openstack.org,resources=controlplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controlplane.openstack.org,resources=controlplanes/status,verbs=get;update;patch

// Reconcile - controleplane api
func (r *ControlPlaneReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("controlplane", req.NamespacedName)

	// Fetch the ControlPlane instance
	instance := &controlplanev1beta1.ControlPlane{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	setDefaults(instance)

	data, err := getRenderData(context.TODO(), r.Client, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	objs := []*uns.Unstructured{}

	// Generate the MariaDB objects
	manifests, err := bindatautil.RenderDir(filepath.Join(ManifestPath, "mariadb"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render mariadb manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the AMQ Interconnect objects
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "interconnect"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render interconnect manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Keystone objects
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "keystone"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render keystone manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Glance objects
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "glance"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render glance manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Placement objects
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "placement"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render placement manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Neutron objects
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "neutron"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render neutron manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Cinder objects
	// TODO: how to handle adding additional cinder-volume services using openstack-cluster-operator
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "cinder"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render cinder manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Generate the Nova objects
	// TODO: how to handle adding additional cells using openstack-cluster-operator
	manifests, err = bindatautil.RenderDir(filepath.Join(ManifestPath, "nova"), &data)
	if err != nil {
		ctrl.Log.Error(err, "Failed to render nova manifests : %v")
		return ctrl.Result{}, err
	}
	objs = append(objs, manifests...)

	// Apply the objects to the cluster
	oref := metav1.NewControllerRef(instance, instance.GroupVersionKind())
	labelSelector := map[string]string{
		ownerUIDLabelSelector:       string(instance.UID),
		ownerNameSpaceLabelSelector: instance.Namespace,
		ownerNameLabelSelector:      instance.Name,
	}
	for _, obj := range objs {
		// Set owner reference on objects in the same namespace as the operator
		if obj.GetNamespace() == instance.Namespace {
			obj.SetOwnerReferences([]metav1.OwnerReference{*oref})
		}
		// merge owner ref label into labels on the objects
		obj.SetLabels(labels.Merge(obj.GetLabels(), labelSelector))
		objs = append(objs, obj)

		if err := bindatautil.ApplyObject(context.TODO(), r.Client, obj); err != nil {
			ctrl.Log.Error(err, "Failed to apply objects")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager -
func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&controlplanev1beta1.ControlPlane{}).
		Complete(r)
}

func getRenderData(ctx context.Context, client client.Client, instance *controlplanev1beta1.ControlPlane) (bindatautil.RenderData, error) {
	data := bindatautil.MakeRenderData()
	data.Data["KeystoneReplicas"] = instance.Spec.Keystone.Replicas
	data.Data["GlanceReplicas"] = instance.Spec.Glance.Replicas
	data.Data["PlacementReplicas"] = instance.Spec.Placement.Replicas
	data.Data["InterconnectReplicas"] = instance.Spec.Interconnect.Replicas
	data.Data["NovaAPIReplicas"] = instance.Spec.Nova.NovaAPIReplicas
	data.Data["NovaConductorReplicas"] = instance.Spec.Nova.NovaConductorReplicas
	data.Data["NovaMetadataReplicas"] = instance.Spec.Nova.NovaMetadataReplicas
	data.Data["NovaNoVNCProxyReplicas"] = instance.Spec.Nova.NovaNoVNCProxyReplicas
	data.Data["NovaSchedulerReplicas"] = instance.Spec.Nova.NovaSchedulerReplicas
	data.Data["CinderAPIReplicas"] = instance.Spec.Cinder.CinderAPIReplicas
	data.Data["CinderBackupReplicas"] = instance.Spec.Cinder.CinderBackupReplicas
	data.Data["CinderSchedulerReplicas"] = instance.Spec.Cinder.CinderSchedulerReplicas
	data.Data["CinderVolumeReplicas"] = instance.Spec.Cinder.CinderVolumeReplicas
	data.Data["NeutronAPIReplicas"] = instance.Spec.Neutron.Replicas
	data.Data["Namespace"] = instance.Namespace
	data.Data["StorageClass"] = instance.Spec.StorageClass
	return data, nil
}

func setDefaults(instance *controlplanev1beta1.ControlPlane) {
	// required to be greated than 0 by the interconnect operator
	if instance.Spec.Interconnect.Replicas < 1 {
		instance.Spec.Interconnect.Replicas = 1
	}
}
