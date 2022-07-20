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

	logger.Info("Reading credentials...")
	username := space.Spec.Connection.Credentials.Username
	secret := &corev1.Secret{}
	passwordRef := space.Spec.Connection.Credentials.PasswordRef
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: passwordRef}, secret)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading credentials: %w", err)
	}
	password := string(secret.Data[username])

	logger.Info("Reading current spaces...")
	var currentSpaces []Space
	url := space.Spec.Connection.URL + "/api/spaces/space"
	err = GetRequest(logger, url, username, password, nil, &currentSpaces)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading current spaces error: %w", err)
	}
	currentSpaceIdToSpace := make(map[string]Space)
	for _, currentSpace := range currentSpaces {
		if strings.HasPrefix(currentSpace.Id, prefix) && strings.HasSuffix(currentSpace.Id, suffix) {
			currentSpaceIdToSpace[currentSpace.Id] = currentSpace
		}
	}
	logger.Info("Found <" + strconv.Itoa(len(currentSpaces)) + "> spaces")
	logger.Info("Found <" + strconv.Itoa(len(currentSpaceIdToSpace)) + "> matching spaces")

	logger.Info("Synchronizing current/expected spaces...")
	expectedSpaces := space.Spec.Spaces
	for _, expectedSpace := range expectedSpaces {
		spaceId := GetFullName(prefix, expectedSpace.Id, suffix)
		logger.Info("Processing space <" + spaceId + ">...")
		if _, ok := currentSpaceIdToSpace[spaceId]; ok {
			logger.Info("Updating current space <" + spaceId + ">...")
			_, err = UpdateSpace(logger, url, username, password, spaceId, expectedSpace)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while updating space: %w", err)
			}
		} else {
			logger.Info("Creating expected space <" + spaceId + ">...")
			_, err = CreateSpace(logger, url, username, password, spaceId, expectedSpace)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while creating space: %w", err)
			}
		}
	}

	// TODO set status
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanav1alpha1.Space{}).
		Complete(r)
}

func CreateSpace(logger logr.Logger, url string, username string, password string, spaceId string, spaceSpec kibanav1alpha1.KibanaSpace) ([]byte, error) {
	space := Space{}
	space.Id = spaceId
	space.Name = spaceSpec.Name
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
	space := Space{}
	space.Id = spaceId
	space.Name = spaceId
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

type Space struct {
	Id               string   `json:"id,omitempty"`
	Name             string   `json:"name,omitempty"`
	Description      string   `json:"description,omitempty"`
	DisabledFeatures []string `json:"disabledFeatures,omitempty"`
}
