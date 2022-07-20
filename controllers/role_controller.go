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
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"

	kibanav1alpha1 "k8s.svketen.dev/api/v1alpha1"
)

// RoleReconciler reconciles a Role object
type RoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=roles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=roles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kibana.k8s.svketen.dev,resources=roles/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *RoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := log.Log.WithValues("role", req.NamespacedName)
	logger.Info("Role reconcile method...")

	// fetch the role CR instance
	role := &kibanav1alpha1.Role{}
	err := r.Get(ctx, req.NamespacedName, role)
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
	if !role.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Ignoring since resource is being deleted.", "name", req.NamespacedName)

		return ctrl.Result{}, nil
	}

	prefix := role.Spec.Prefix
	suffix := role.Spec.Suffix
	namespace := role.Namespace

	logger.Info("Reading credentials...")
	username := role.Spec.Connection.Credentials.Username
	secret := &corev1.Secret{}
	passwordRef := role.Spec.Connection.Credentials.PasswordRef
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: passwordRef}, secret)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading credentials: %w", err)
	}
	password := string(secret.Data[username])

	logger.Info("Reading current roles...")
	var currentRoles []Role
	url := role.Spec.Connection.URL + "/api/security/role"
	err = GetRequest(logger, url, username, password, nil, &currentRoles)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error while reading current roles error: %w", err)
	}
	currentRoleNameToRole := make(map[string]Role)
	for _, currentRole := range currentRoles {
		if strings.HasPrefix(currentRole.Name, prefix) && strings.HasSuffix(currentRole.Name, suffix) {
			currentRoleNameToRole[currentRole.Name] = currentRole
		}
	}
	logger.Info("Found <" + strconv.Itoa(len(currentRoles)) + "> roles")
	logger.Info("Found <" + strconv.Itoa(len(currentRoleNameToRole)) + "> matching roles")

	logger.Info("Synchronizing current/expected roles...")
	expectedRoles := role.Spec.Roles
	for _, expectedRole := range expectedRoles {
		roleName := GetRoleName(prefix, expectedRole.Name, suffix)
		logger.Info("Processing role <" + roleName + ">...")
		if _, ok := currentRoleNameToRole[roleName]; ok {
			logger.Info("Updating current role <" + roleName + ">...")
			_, err = CreateOrUpdateRole(logger, url, username, password, roleName, expectedRole)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while updating role: %w", err)
			}
		} else {
			logger.Info("Creating expected role <" + roleName + ">...")
			_, err = CreateOrUpdateRole(logger, url, username, password, roleName, expectedRole)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error while creating role: %w", err)
			}
		}
	}

	// TODO set status
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kibanav1alpha1.Role{}).
		Complete(r)
}

func CreateOrUpdateRole(logger logr.Logger, url string, username string, password string, roleName string, roleSpec kibanav1alpha1.KibanaRole) ([]byte, error) {
	role := Role{}
	role.Kibana = roleSpec.Kibana
	role.Elasticsearch = roleSpec.Elasticsearch

	body, err := json.Marshal(role)
	if err != nil {
		return nil, err
	}

	result, err := PutRequest(logger, url+"/"+roleName, username, password, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetRequest(logger logr.Logger, url string, username string, password string, body io.Reader, result any) error {
	logger.Info("Sending GET to <" + url + ">")
	// url := URL + "/api/security/role"
	request, err := http.NewRequest("GET", url, body)
	if err != nil {
		return err
	}

	bodyText, err := SendRequest(logger, request, username, password)
	if err != nil {
		return err
	}

	return json.Unmarshal(bodyText, result)
}

func PutRequest(logger logr.Logger, url string, username string, password string, body io.Reader) ([]byte, error) {
	logger.Info("Sending PUT to <" + url + ">")

	request, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	bodyText, err := SendRequest(logger, request, username, password)
	if err != nil {
		return nil, err
	}

	return bodyText, nil
}

func SendRequest(logger logr.Logger, request *http.Request, username string, password string) ([]byte, error) {
	logger.Info("Sending request...")

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("kbn-xsrf", "true")
	request.SetBasicAuth(username, password)

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	// TODO handle status code 4xx 5xx
	logger.Info("Response <" + response.Status + ">")

	bodyText, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return bodyText, nil
}

func GetRoleName(prefix string, name string, suffix string) string {
	return prefix + name + suffix
}

type Role struct {
	Name          string                       `json:"name,omitempty"`
	Elasticsearch kibanav1alpha1.Elasticsearch `json:"elasticsearch,omitempty"`
	Kibana        []kibanav1alpha1.Kibana      `json:"kibana,omitempty"`
}
