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
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	controlplanev1beta1 "github.com/openstack-k8s-operators/openstack-cluster-operator/api/v1beta1"
)

// OpenStackClientReconciler reconciles a OpenStackClient object
type OpenStackClientReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=controlplane.openstack.org,resources=openstackclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controlplane.openstack.org,resources=openstackclients/status,verbs=get;update;patch

// Reconcile OpenStackClient requests
func (r *OpenStackClientReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("openstackclient", req.NamespacedName)

	instance := &controlplanev1beta1.OpenStackClient{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)
	r.Log.Info("OpenStackClient values", "Name", instance.Name, "Namespace", instance.Namespace, "Secret", instance.Spec.OpenStackConfigSecret, "Spec", fmt.Sprintf("%T", instance.Spec), "Image", instance.Spec.ContainerImage)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	err = r.reconcileDeployment(instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager func
func (r *OpenStackClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&controlplanev1beta1.OpenStackClient{}).
		Complete(r)
}

func (r *OpenStackClientReconciler) reconcileDeployment(instance *controlplanev1beta1.OpenStackClient) error {
	clientDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	r.Log.Info("openstack-config-secret name", "Name", instance.Spec.OpenStackConfigSecret)
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, clientDeployment, func() error {
		clientDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "openstack-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: instance.Spec.OpenStackConfigMap,
						},
					},
				},
			},
			{
				Name: "openstack-config-secret",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: instance.Spec.OpenStackConfigSecret,
					},
				},
			},
		}

		labels := map[string]string{
			"app": "openstackclient",
		}
		clientDeployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: labels,
		}
		var replicas int32 = 1
		clientDeployment.Spec.Replicas = &replicas
		clientDeployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		}
		clientDeployment.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:    "openstackclient",
				Image:   instance.Spec.ContainerImage,
				Command: []string{"sleep", "infinity"},
				Env: []corev1.EnvVar{
					{
						Name:  "OS_CLOUD",
						Value: "default",
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "openstack-config",
						MountPath: "/etc/openstack/clouds.yaml",
						SubPath:   "clouds.yaml",
					},
					{
						Name:      "openstack-config-secret",
						MountPath: "/etc/openstack/secure.yaml",
						SubPath:   "secure.yaml",
					},
				},
			},
		}

		return nil
	})

	return err
}
