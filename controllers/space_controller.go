/*
Copyright 2022 svketen.

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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kibanav1alpha1 "k8s.svketen.dev/api/v1alpha1"
)

// SpaceReconciler reconciles a Space object
type SpaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=spaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=spaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=spaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *SpaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("role", req.NamespacedName)
	logger.Info("Space reconcile method...")

	// fetch the role CR instance
	space := &kibanav1alpha1.Space{}
	err := r.Get(ctx, req.NamespacedName, space)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("Resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("get role CR error: %w", err)
	}

	// Check if it is being deleted
	if !space.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Ignoring since resource is being deleted.", "name", req.NamespacedName)

		return ctrl.Result{}, nil
	}

	prefix := space.Spec.Prefix
	suffix := space.Spec.Suffix
	namespace := space.Namespace
	config := space.Spec.Config
	expectedValues := space.Spec.Spaces
	var actualValues []kibanav1alpha1.KibanaSpace
	url := config.Connection.URL + "/api/spaces/space"
	actualKeyToValue := make(map[string]kibanav1alpha1.KibanaSpace)

	// TODO way to much duplicated code
	logger.Info("Reading credentials...")
	username := config.Connection.Credentials.Username
	secret := &corev1.Secret{}
	passwordRef := config.Connection.Credentials.PasswordRef
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: passwordRef}, secret)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading credentials: %w", err)
	}
	password := string(secret.Data[username])

	logger.Info("Reading current values...")
	err = GetRequest(logger, url, username, password, nil, &actualValues)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading current values error: %w", err)
	}
	logger.Info("Found <" + strconv.Itoa(len(actualValues)) + "> values")

	for _, currentSpace := range actualValues {
		if strings.HasPrefix(currentSpace.Id, prefix) && strings.HasSuffix(currentSpace.Id, suffix) {
			actualKeyToValue[currentSpace.Id] = currentSpace
		}
	}
	logger.Info("Found <" + strconv.Itoa(len(actualKeyToValue)) + "> matching values")

	var created int32 = 0
	var updated int32 = 0
	var deleted int32 = 0
	logger.Info("Synchronizing current/expected values...")
	for _, expectedValue := range expectedValues {
		id := GetFullName(prefix, expectedValue.Id, suffix)
		logger.Info("Processing value <" + id + ">...")
		if _, ok := actualKeyToValue[id]; ok {
			logger.Info("Updating current value <" + id + ">...")
			actualValue := actualKeyToValue[id]
			if !kibanav1alpha1.IsKibanaSpaceEqual(actualValue, expectedValue) {
				logger.Info("Updating current value <" + id + ">...")
				_, err = UpdateSpace(logger, url, username, password, id, expectedValue)
				if err != nil {
					return ctrl.Result{}, fmt.Errorf("error while updating value: %w", err)
				}
				updated += 1
			} else {
				logger.Info("Skipped update because actual value as specified <" + id + ">.")
			}
		} else {
			logger.Info("Creating expected space <" + id + ">...")
			_, err = CreateSpace(logger, url, username, password, id, expectedValue)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while creating value: %w", err)
			}
			created += 1
		}
		delete(actualKeyToValue, id)
	}

	// deletion is only possible if prefix or suffix is given
	if len(prefix)+len(suffix) > 0 {
		logger.Info("Found <" + strconv.Itoa(len(actualKeyToValue)) + "> outdated values")
		for id, _ := range actualKeyToValue {
			logger.Info("Deleting current value <" + id + ">...")
			if space.Spec.Config.Delete {
				_, err := DeleteRequest(logger, url+"/"+id, username, password)
				if err != nil {
					return ctrl.Result{}, err
				}
				updated += 1
			} else {
				logger.Info("Deletion of outdated values is deactivated")
			}
		}
	}

	// Update status if needed
	if (created + updated + deleted) > 0 {
		space.Status.Created += created
		space.Status.Updated += updated
		space.Status.Deleted += deleted
		err := r.Status().Update(ctx, space)
		if err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanav1alpha1.Space{}).
		Complete(r)
}

func CreateSpace(logger logr.Logger, url string, username string, password string, spaceId string, spaceSpec kibanav1alpha1.KibanaSpace) ([]byte, error) {
	space := kibanav1alpha1.KibanaSpace{}
	space.Id = spaceId
	space.Name = spaceSpec.Name
	space.Description = spaceSpec.Description
	space.DisabledFeatures = spaceSpec.DisabledFeatures

	body, err := json.Marshal(space)
	if err != nil {
		return nil, err
	}

	result, err := PostRequest(logger, url, username, password, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func UpdateSpace(logger logr.Logger, url string, username string, password string, spaceId string, spaceSpec kibanav1alpha1.KibanaSpace) ([]byte, error) {
	space := kibanav1alpha1.KibanaSpace{}
	space.Id = spaceId
	space.Name = spaceSpec.Name
	space.Description = spaceSpec.Description
	space.DisabledFeatures = spaceSpec.DisabledFeatures

	body, err := json.Marshal(space)
	if err != nil {
		return nil, err
	}

	result, err := PutRequest(logger, url+"/"+spaceId, username, password, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return result, nil
}
