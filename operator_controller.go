/*
Copyright 2022.

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
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/keith-cullen/operator/api/v1alpha1"
)

// OperatorReconciler reconciles a Operator object
type OperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.example.org,resources=operators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.example.org,resources=operators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.example.org,resources=operators/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Operator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *OperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Operator instance
	operator := &cachev1alpha1.Operator{}
	err := r.Get(ctx, req.NamespacedName, operator)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Operator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Operator resource")
		return ctrl.Result{}, err
	}
	log.Info("Reconciling Operator Name: %s, Namespace: %s", operator.Name, operator.Namespace)

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: operator.Name, Namespace: operator.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForOperator(operator)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		log.Info("Created new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := operator.Spec.Size
	if *found.Spec.Replicas != size {
		str := fmt.Sprintf("Updating deployment. Size: expected: %v, actual: %v", size, *found.Spec.Replicas)
		log.Info(str)
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods to be created on the cluster side and the opperand be able
		// to do the next update step accurately.
		str = fmt.Sprintf("Updated deployment. Size: expected: %v, actual: %v", size, *found.Spec.Replicas)
		log.Info(str)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Update the Operator status with the pod names
	// List the pods for this operators's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(operator.Namespace),
		client.MatchingLabels(labelsForOperator(operator.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "Failed to list pods", "Operator.Namespace", operator.Namespace, "Operator.Name", operator.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Pods if needed
	if !reflect.DeepEqual(podNames, operator.Status.Pods) {
		operator.Status.Pods = podNames
		err := r.Status().Update(ctx, operator)
		if err != nil {
			log.Error(err, "Failed to update Operator status")
			return ctrl.Result{}, err
		}
		log.Info("Operator status updated")
	}

	return ctrl.Result{}, nil
}

// deploymentForOperator returns a Operator Deployment object
func (r *OperatorReconciler) deploymentForOperator(op *cachev1alpha1.Operator) *appsv1.Deployment {
	ls := labelsForOperator(op.Name)
	replicas := op.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      op.Name,
			Namespace: op.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					// Ensure restrictive standard for the Pod.
					SecurityContext: &corev1.PodSecurityContext{
						//                                              RunAsNonRoot: &[]bool{true}[0],
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image: "localhost:5000/operatorwebapp:latest",
						Name:  "operatorwebapp",
						SecurityContext: &corev1.SecurityContext{
							//                                                      RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"All",
								},
							},
						},
						Command: []string{"lighttpd", "-D", "-f", "lighttpd.conf"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8081,
							Name:          "operatorwebapp",
						}},
					}},
				},
			},
		},
	}
	// Set Operator instance as the owner and controller
	ctrl.SetControllerReference(op, dep, r.Scheme)
	return dep
}

// labelsForOperator returns the labels for selecting the resources
// belonging to the given Operator CR name.
func labelsForOperator(name string) map[string]string {
	return map[string]string{"app": "operator", "operator_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Operator{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
