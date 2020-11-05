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
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	logger := r.Log.WithValues("namespace", req.Namespace, "at", req.NamespacedName)
	logger.Info("=== Reconciling At")
	// Fetch the at instance
	instance := &cnatv1alpha1.At{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// could have been deleted after run therefore do not requie
			return ctrl.Result{}, nil
		}
		// Error reading the object therefore requeues the request.
		return ctrl.Result{}, err
	}
	// your logic here
	if instance.Status.Phase == "" {
		// set the instance of pending if the state is blank
		instance.Status.Phase = cnatv1alpha1.PhasePending
	}
	switch instance.Status.Phase {
	case cnatv1alpha1.PhasePending:
		logger.Info("Phase: Pending")
		// check if it is time to act
		logger.Info("Checking status Target " + instance.Spec.Schedule)
		d, err := timeUntilSchedule(instance.Spec.Schedule)
		if err != nil {
			logger.Error(err, "Schedule parsing failure")
			return ctrl.Result{}, err
		}
		logger.Info(fmt.Sprintf("Schedule passed successfully d: %v\n", d))
		if d > 0 {
			// not yet time therefore return
			return ctrl.Result{RequeueAfter: d}, nil
		}
		logger.Info("It's time!", "Ready to execute", instance.Spec.Command)
		instance.Status.Phase = cnatv1alpha1.PhaseRunning
	case cnatv1alpha1.PhaseRunning:
		logger.Info("Phase: Running")
		pod := newPodForCR(instance)
		err := controllerutil.SetControllerReference(instance, pod, r.Scheme)
		if err != nil {
			// requeues with error
			return ctrl.Result{}, err
		}
		found := &corev1.Pod{}
		nsName := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
		err = r.Get(context.TODO(), nsName, found)
		// if the pod already exists if not then create the pod in one shot
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), pod)
			if err != nil {
				// requeues with error
				return ctrl.Result{}, err
			}
			logger.Info("Pod Launched", "name", pod.Name)
		} else if err != nil {
			// requeues - as error occurs so we need to requeues.
			return ctrl.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			logger.Info("Container terminated", "reason", found.Status.Reason, "message", found.Status.Message)
			instance.Status.Phase = cnatv1alpha1.PhaseDone
		} else {
			// don't requeues as the requeues will happen as the stauts changes
			return ctrl.Result{}, nil
		}
	case cnatv1alpha1.PhaseDone:
		logger.Info("Phase Done")
		return ctrl.Result{}, nil
	default:
		logger.Info("NOP")
		return ctrl.Result{}, nil
	}

	// Update the instance, settingo the right phase
	logger.Info(fmt.Sprintf("Updating the instance %v with status %s", instance.Name, instance.Status.Phase))
	//r.Status().Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption)
	err = r.Update(context.TODO(), instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Don't requeues
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
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: strings.Split(cr.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}

}

func timeUntilSchedule(schedule string) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:00Z"
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
