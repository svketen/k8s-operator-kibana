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
	"encoding/json"
	"github.com/go-logr/logr"
	"io"
	"io/ioutil"
	"net/http"
)

func GetFullName(prefix string, name string, suffix string) string {
	return prefix + name + suffix
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

func PostRequest(logger logr.Logger, url string, username string, password string, body io.Reader) ([]byte, error) {
	logger.Info("Sending POST to <" + url + ">")

	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	bodyText, err := SendRequest(logger, request, username, password)
	if err != nil {
		return nil, err
	}

	return bodyText, nil
}

func DeleteRequest(logger logr.Logger, url string, username string, password string) ([]byte, error) {
	logger.Info("Sending DELETE to <" + url + ">")

	request, err := http.NewRequest("DELETE", url, nil)
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
