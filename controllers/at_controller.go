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
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/ctrl"

	cnatv1alpha1 "github.com/chauhanr/cnat-operator/api/v1alpha1"
)

// AtReconciler ctrls a At object
type AtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cnat.programming-k8s.info,resources=ats,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cnat.programming-k8s.info,resources=ats/status,verbs=get;update;patch

func (r *AtReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//todo = context.Background()
	logger := r.Log.WithValues("at", req.NamespacedName)
	logger.Info("=== Reconciling At")
	// Fetch the at instance
	instance := &cnatv1alpha1.At{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// could have been deleted after run therefore do not requie
			return ctrl.Result{}, nil
		}
		// Error reading the object therefore requeue the request.
		return ctrl.Result{}, err
	}
	// your logic here

	return ctrl.Result{}, nil
}

func newPodForCR(cr *cnatv1alpha1.At) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				Name:    "busybox",
				Image:   "busybox",
				Command: strings.Split(cr.Spec.Command, " "),
			},
		},
		RestartPolicy: corev1.RestartPolicyOnFailure,
	}

}

func timeUntilSchedule(schedule string) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:02Z"
	s, err := time.Parse(layout, schedule)
	if err != nil {
		return time.Duration(0), err
	}
	return s.Sub(now), nil
}

func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnatv1alpha1.At{}).
		Complete(r)
}
