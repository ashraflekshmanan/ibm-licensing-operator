//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/ibm/ibm-licensing-operator/api/v1alpha1"
	res "github.com/ibm/ibm-licensing-operator/controllers/resources"
	"github.com/ibm/ibm-licensing-operator/controllers/resources/service"
	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconcileLSFunctionType = func(*operatorv1alpha1.IBMLicensing) (reconcile.Result, error)

func (r *IBMLicensingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := res.UpdateCacheClusterExtensions(mgr.GetAPIReader()); err != nil {
		r.Log.Error(err, "Error during checking K8s API")
	}

	watcher := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.IBMLicensing{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{})

	if res.IsRouteAPI {
		watcher.Owns(&operatorv1alpha1.IBMLicenseServiceReporter{})
	}

	return watcher.Complete(r)
}

// blank assignment to verify that IBMLicensingReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &IBMLicensingReconciler{}

// IBMLicensingReconciler reconciles a IBMLicensing object
type IBMLicensingReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client.Client
	client.Reader
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// //kubebuilder:rbac:namespace=ibm-common-services,groups=,resources=pod,verbs=get;list;watch;create;update;patch;delete

// Reconcile reads that state of the cluster for a IBMLicensing object and makes changes based on the state read
// and what is in the IBMLicensing.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// +kubebuilder:rbac:namespace=ibm-common-services,groups=operator.ibm.com,resources=ibmlicensings;ibmlicensings/status;ibmlicensings/finalizers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups="apps",resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:namespace=ibm-common-services,groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create;watch;list
// +kubebuilder:rbac:namespace=ibm-common-services,groups="",resources=pods,verbs=get
// +kubebuilder:rbac:namespace=ibm-common-services,groups="",resources=pods,verbs=get
// +kubebuilder:rbac:namespace=ibm-common-services,groups=apps,resources=replicasets;deployments,verbs=get
// +kubebuilder:rbac:namespace=ibm-common-services,groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups="",resources=pods;nodes;namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups=networking.k8s.io;extensions,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-common-services,groups="",resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets;namespaces;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.openshift.io,resources=servicecas,verbs=list
// +kubebuilder:rbac:groups=operator.ibm.com,resources=ibmlicensings;ibmlicensings/status;ibmlicensings/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *IBMLicensingReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {

	reqLogger := r.Log.WithValues("ibmlicensing", req.NamespacedName)
	reqLogger.Info("Reconciling IBMLicensing")

	if err := res.UpdateCacheClusterExtensions(r.Reader); err != nil {
		reqLogger.Error(err, "Error during checking K8s API")
	}

	r.controllerStatus()

	// Fetch the IBMLicensing instance
	foundInstance := &operatorv1alpha1.IBMLicensing{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, foundInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			// reqLogger.Info("IBMLicensing resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the req.
		// reqLogger.Error(err, "Failed to get IBMLicensing")
		return reconcile.Result{}, err
	}
	instance := foundInstance.DeepCopy()

	err = service.UpdateVersion(r.Client, instance)
	if err != nil {
		reqLogger.Error(err, "Can not update version in CR")
	}

	err = instance.Spec.FillDefaultValues(res.IsServiceCAAPI, res.IsRouteAPI, res.RHMPEnabled)
	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("got IBM License Service application, version=" + instance.Spec.Version)

	var recResult reconcile.Result

	reconcileFunctions := []interface{}{
		r.reconcileAPISecretToken,
		r.reconcileUploadToken,
		r.reconcileUploadConfigMap,
		r.reconcileServices,
		r.reconcileDeployment,
		r.reconcileIngress,
		r.reconcileRoute,
	}

	if instance.Spec.IsRHMPEnabled() {
		reconcileFunctions = append(reconcileFunctions, r.reconcileServiceMonitor, r.reconcileNetworkPolicy)
	}

	for _, reconcileFunction := range reconcileFunctions {
		recResult, err = reconcileFunction.(reconcileLSFunctionType)(instance)
		if err != nil || recResult.Requeue {
			return recResult, err
		}
	}

	// Update status logic, using foundInstance, because we do not want to add filled default values to yaml
	return r.updateStatus(foundInstance, reqLogger)
}

func (r *IBMLicensingReconciler) updateStatus(instance *operatorv1alpha1.IBMLicensing, reqLogger logr.Logger) (reconcile.Result, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Spec.InstanceNamespace),
		client.MatchingLabels(service.LabelsForLicensingPod(instance)),
	}
	if err := r.Client.List(context.TODO(), podList, listOpts...); err != nil {
		reqLogger.Error(err, "Failed to list pods")
		return reconcile.Result{}, err
	}

	var podStatuses []corev1.PodStatus
	for _, pod := range podList.Items {
		if pod.Status.Conditions != nil {
			i := 0
			for _, podCondition := range pod.Status.Conditions {
				if (podCondition.LastProbeTime == metav1.Time{Time: time.Time{}}) {
					// Time{} is treated as null and causes error at status update so value need to be changed to some other default empty value
					pod.Status.Conditions[i].LastProbeTime = metav1.Time{
						Time: time.Unix(0, 1),
					}
				}
				i++
			}
		}
		podStatuses = append(podStatuses, pod.Status)
	}

	if !reflect.DeepEqual(podStatuses, instance.Status.LicensingPods) {
		reqLogger.Info("Updating IBMLicensing status")
		instance.Status.LicensingPods = podStatuses
		err := r.Client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Info("Warning: Failed to update pod status, this does not affect License Service")
		}
	}

	reqLogger.Info("reconcile all done")
	return reconcile.Result{}, nil
}

func (r *IBMLicensingReconciler) reconcileAPISecretToken(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	reqLogger := r.Log.WithValues("reconcileAPISecretToken", "Entry", "instance.GetName()", instance.GetName())
	expectedSecret, err := service.GetAPISecretToken(instance)
	if err != nil {
		reqLogger.Info("Failed to get expected secret")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Minute,
		}, err
	}
	foundSecret := &corev1.Secret{}
	return r.reconcileResourceNamespacedExistence(instance, expectedSecret, foundSecret)
}

func (r *IBMLicensingReconciler) reconcileUploadToken(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	reqLogger := r.Log.WithValues("reconcileUploadToken", "Entry", "instance.GetName()", instance.GetName())
	expectedSecret, err := service.GetUploadToken(instance)
	if err != nil {
		reqLogger.Info("Failed to get expected secret")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Minute,
		}, err
	}
	foundSecret := &corev1.Secret{}
	return r.reconcileResourceNamespacedExistence(instance, expectedSecret, foundSecret)
}

func (r *IBMLicensingReconciler) reconcileUploadConfigMap(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	reqLogger := r.Log.WithValues("reconcileUploadConfigMap", "Entry", "instance.GetName()", instance.GetName())
	expectedCM := service.GetUploadConfigMap(instance)
	foundCM := &corev1.ConfigMap{}
	reconcileResult, err := r.reconcileResourceNamespacedExistence(instance, expectedCM, foundCM)
	if err != nil || reconcileResult.Requeue {
		return reconcileResult, err
	}
	if foundCM.Data[service.UploadConfigMapKey] == expectedCM.Data[service.UploadConfigMapKey] {
		return reconcile.Result{}, nil
	}
	return res.UpdateResource(&reqLogger, r.Client, expectedCM, foundCM)
}

func (r *IBMLicensingReconciler) reconcileServices(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	var (
		result reconcile.Result
		err    error
	)
	reqLogger := r.Log.WithValues("reconcileServices", "Entry", "instance.GetName()", instance.GetName())
	expectedServices := service.GetServices(instance)
	foundService := &corev1.Service{}
	for _, es := range expectedServices {
		result, err = r.reconcileResourceNamespacedExistence(instance, es, foundService)
		if err != nil || result.Requeue {
			return result, err
		}
		result, err = res.UpdateServiceIfNeeded(&reqLogger, r.Client, es, foundService)
	}

	return result, err
}

func (r *IBMLicensingReconciler) reconcileServiceMonitor(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	if !instance.Spec.IsRHMPEnabled() {
		return reconcile.Result{}, nil
	}
	reqLogger := r.Log.WithValues("reconcileServiceMonitor", "Entry", "instance.GetName()", instance.GetName())
	expectedServiceMonitor := service.GetServiceMonitor(instance)
	foundServiceMonitor := &monitoringv1.ServiceMonitor{}
	result, err := r.reconcileResourceNamespacedExistence(instance, expectedServiceMonitor, foundServiceMonitor)
	if err != nil || result.Requeue {
		return result, err
	}
	result, err = res.UpdateServiceMonitor(&reqLogger, r.Client, expectedServiceMonitor, foundServiceMonitor)

	return result, err
}

func (r *IBMLicensingReconciler) reconcileNetworkPolicy(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	if !instance.Spec.IsRHMPEnabled() {
		return reconcile.Result{}, nil
	}
	reqLogger := r.Log.WithValues("reconcileNetworkPolicy", "Entry", "instance.GetName()", instance.GetName())
	expected := service.GetNetworkPolicy(instance)
	found := &extensionsv1.NetworkPolicy{}
	result, err := r.reconcileResourceNamespacedExistence(instance, expected, found)
	if err != nil || result.Requeue {
		return result, err
	}
	result, err = res.UpdateResource(&reqLogger, r.Client, expected, found)

	return result, err
}

func (r *IBMLicensingReconciler) reconcileDeployment(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	reqLogger := r.Log.WithValues("reconcileDeployment", "Entry", "instance.GetName()", instance.GetName())
	expectedDeployment := service.GetLicensingDeployment(instance)

	foundDeployment := &appsv1.Deployment{}
	reconcileResult, err := r.reconcileResourceNamespacedExistence(instance, expectedDeployment, foundDeployment)
	if err != nil || reconcileResult.Requeue {
		return reconcileResult, err
	}

	shouldUpdate := res.ShouldUpdateDeployment(
		&reqLogger,
		&expectedDeployment.Spec.Template,
		&foundDeployment.Spec.Template,
	)
	if shouldUpdate {
		return res.UpdateResource(&reqLogger, r.Client, expectedDeployment, foundDeployment)
	}

	return reconcile.Result{}, nil
}

func (r *IBMLicensingReconciler) reconcileRoute(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	if res.IsRouteAPI && instance.Spec.IsRouteEnabled() {
		expectedRoute := service.GetLicensingRoute(instance)
		foundRoute := &routev1.Route{}
		reconcileResult, err := r.reconcileResourceNamespacedExistence(instance, expectedRoute, foundRoute)
		if err != nil || reconcileResult.Requeue {
			return reconcileResult, err
		}
		reqLogger := r.Log.WithValues("reconcileRoute", "Entry", "instance.GetName()", instance.GetName())
		possibleUpdateNeeded := true
		if foundRoute.ObjectMeta.Name != expectedRoute.ObjectMeta.Name {
			reqLogger.Info("Names not equal", "old", foundRoute.ObjectMeta.Name, "new", expectedRoute.ObjectMeta.Name)
		} else if foundRoute.Spec.To.Name != expectedRoute.Spec.To.Name {
			reqLogger.Info("Specs To Name not equal",
				"old", fmt.Sprintf("%v", foundRoute.Spec),
				"new", fmt.Sprintf("%v", expectedRoute.Spec))
		} else if foundRoute.Spec.TLS == nil && expectedRoute.Spec.TLS != nil {
			reqLogger.Info("Found Route has empty TLS options, but Expected Route has not empty TLS options",
				"old", fmt.Sprintf("%v", foundRoute.Spec.TLS),
				"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS))
		} else if foundRoute.Spec.TLS != nil && expectedRoute.Spec.TLS == nil {
			reqLogger.Info("Expected Route has empty TLS options, but Found Route has not empty TLS options",
				"old", fmt.Sprintf("%v", foundRoute.Spec.TLS),
				"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS))
		} else if foundRoute.Spec.TLS != nil && expectedRoute.Spec.TLS != nil &&
			(foundRoute.Spec.TLS.Termination != expectedRoute.Spec.TLS.Termination ||
				foundRoute.Spec.TLS.InsecureEdgeTerminationPolicy != expectedRoute.Spec.TLS.InsecureEdgeTerminationPolicy) {
			reqLogger.Info("Expected Route has different TLS options than Found Route",
				"old", fmt.Sprintf("%v", foundRoute.Spec.TLS),
				"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS))
		} else {
			possibleUpdateNeeded = false
		}
		if possibleUpdateNeeded {
			return res.UpdateResource(&reqLogger, r.Client, expectedRoute, foundRoute)
		}
	}
	return reconcile.Result{}, nil
}

func (r *IBMLicensingReconciler) reconcileIngress(instance *operatorv1alpha1.IBMLicensing) (reconcile.Result, error) {
	if instance.Spec.IsIngressEnabled() {
		expectedIngress := service.GetLicensingIngress(instance)
		foundIngress := &extensionsv1.Ingress{}
		reconcileResult, err := r.reconcileResourceNamespacedExistence(instance, expectedIngress, foundIngress)
		if err != nil || reconcileResult.Requeue {
			return reconcileResult, err
		}
		reqLogger := r.Log.WithValues("reconcileIngress", "Entry", "instance.GetName()", instance.GetName())
		possibleUpdateNeeded := true
		if foundIngress.ObjectMeta.Name != expectedIngress.ObjectMeta.Name {
			reqLogger.Info("Names not equal", "old", foundIngress.ObjectMeta.Name, "new", expectedIngress.ObjectMeta.Name)
		} else if !reflect.DeepEqual(foundIngress.ObjectMeta.Labels, expectedIngress.ObjectMeta.Labels) {
			reqLogger.Info("Labels not equal",
				"old", fmt.Sprintf("%v", foundIngress.ObjectMeta.Labels),
				"new", fmt.Sprintf("%v", expectedIngress.ObjectMeta.Labels))
		} else if !reflect.DeepEqual(foundIngress.ObjectMeta.Annotations, expectedIngress.ObjectMeta.Annotations) {
			reqLogger.Info("Annotations not equal",
				"old", fmt.Sprintf("%v", foundIngress.ObjectMeta.Annotations),
				"new", fmt.Sprintf("%v", expectedIngress.ObjectMeta.Annotations))
		} else if !reflect.DeepEqual(foundIngress.Spec, expectedIngress.Spec) {
			reqLogger.Info("Specs not equal",
				"old", fmt.Sprintf("%v", foundIngress.Spec),
				"new", fmt.Sprintf("%v", expectedIngress.Spec))
		} else {
			possibleUpdateNeeded = false
		}
		if possibleUpdateNeeded {
			return res.UpdateResource(&reqLogger, r.Client, expectedIngress, foundIngress)
		}
	}
	return reconcile.Result{}, nil
}

func (r *IBMLicensingReconciler) reconcileResourceNamespacedExistence(
	instance *operatorv1alpha1.IBMLicensing, expectedRes res.ResourceObject, foundRes runtime.Object) (reconcile.Result, error) {

	namespacedName := types.NamespacedName{Name: expectedRes.GetName(), Namespace: expectedRes.GetNamespace()}
	return r.reconcileResourceExistence(instance, expectedRes, foundRes, namespacedName)
}

func (r *IBMLicensingReconciler) reconcileResourceExistence(
	instance *operatorv1alpha1.IBMLicensing,
	expectedRes res.ResourceObject,
	foundRes runtime.Object,
	namespacedName types.NamespacedName) (reconcile.Result, error) {

	resType := reflect.TypeOf(expectedRes)
	reqLogger := r.Log.WithValues(resType.String(), "Entry", "instance.GetName()", instance.GetName())

	// expectedRes already set before and passed via parameter
	err := controllerutil.SetControllerReference(instance, expectedRes, r.Scheme)
	if err != nil {
		reqLogger.Error(err, "Failed to define expected resource")
		return reconcile.Result{}, err
	}

	// foundRes already initialized before and passed via parameter
	err = r.Client.Get(context.TODO(), namespacedName, foundRes)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info(resType.String()+" does not exist, trying creating new one", "Name", expectedRes.GetName(),
				"Namespace", expectedRes.GetNamespace())
			err = r.Client.Create(context.TODO(), expectedRes)
			if err != nil {
				if !errors.IsAlreadyExists(err) {
					reqLogger.Error(err, "Failed to create new "+resType.String(), "Name", expectedRes.GetName(),
						"Namespace", expectedRes.GetNamespace())
					return reconcile.Result{}, err
				}
			}
			// Created successfully, or already exists - return and requeue
			time.Sleep(time.Second * 5)
			return reconcile.Result{Requeue: true, RequeueAfter: time.Second}, nil
		}
		reqLogger.Error(err, "Failed to get "+resType.String(), "Name", expectedRes.GetName(),
			"Namespace", expectedRes.GetNamespace())
		return reconcile.Result{}, err
	}
	reqLogger.Info(resType.String() + " is correct!")
	return reconcile.Result{}, nil
}

func (r *IBMLicensingReconciler) controllerStatus() {
	if res.IsRouteAPI {
		r.Log.Info("Route feature is enabled")
	} else {
		r.Log.Info("Route feature is disabled")
	}
	if res.IsServiceCAAPI {
		r.Log.Info("ServiceCA feature is enabled")
	} else {
		r.Log.Info("ServiceCA feature is disabled")
	}
	if res.RHMPEnabled {
		r.Log.Info("RHMP is enabled")
	} else {
		r.Log.Info("RHMP is disabled")
	}
}