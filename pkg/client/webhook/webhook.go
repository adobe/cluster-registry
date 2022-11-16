/*
Copyright 2021 Adobe. All rights reserved.
This file is licensed to you under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License. You may obtain a copy
of the License at http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
OF ANY KIND, either express or implied. See the License for the specific language
governing permissions and limitations under the License.
*/

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	configv1 "github.com/adobe/cluster-registry/pkg/api/config/v1"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Server ...
type Server struct {
	Client      client.Client
	Namespace   string
	Log         logr.Logger
	BindAddress string
	Metrics     monitoring.MetricsI
	AlertMap    []configv1.AlertRule
}

const (
	// DeadMansSwitchAlertName is the name of the DMS alert
	DeadMansSwitchAlertName = "CRCDeadMansSwitch"
)

// Start starts the webhook server
func (s *Server) Start() error {
	http.HandleFunc("/webhook", s.webhookHandler)
	server := &http.Server{
		Addr:              s.BindAddress,
		ReadHeaderTimeout: 30 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	var alert Alert

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.Log.Error(err, "unable to read response body")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &alert)
	if err != nil {
		s.Log.Error(err, "unable to unmarshal response body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	s.Log.Info("got alert", "alert", alert)
	err = s.process(alert)
	if err != nil {
		s.Log.Error(err, "unable to handle alert", "alert", alert)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) process(alert Alert) error {
	// DeadMansSwitchAlert should always fire
	if alert.CommonLabels.Alertname == DeadMansSwitchAlertName && alert.Status == AlertStatusFiring {
		s.Metrics.RecordDMSLastTimestamp()
		s.Log.Info("received deadmansswitch", "alertname", DeadMansSwitchAlertName)
		return nil
	}

	for _, a := range s.AlertMap {

		// accept only preconfigured alerts
		if a.AlertName != alert.CommonLabels.Alertname {
			continue
		}

		var tag map[string]string

		if alert.Status == AlertStatusFiring {
			s.Log.Info("OnFiring", "alert", alert.CommonLabels.Alertname, "tag", a.OnFiring)
			tag = a.OnFiring
		} else if alert.Status == AlertStatusResolved {
			s.Log.Info("OnResolved", "alert", alert.CommonLabels.Alertname, "tag", a.OnResolved)
			tag = a.OnResolved
		} else {
			return fmt.Errorf("invalid alert status")
		}

		return retry(s.updateClusterTags, tag, 3)
	}
	return fmt.Errorf("unmapped alert received via webhook")
}

func (s *Server) updateClusterTags(tag map[string]string) error {

	clusterList := &registryv1.ClusterList{}
	err := s.Client.List(context.TODO(), clusterList, &client.ListOptions{Namespace: s.Namespace})
	if err != nil {
		return err
	}

	for i := range clusterList.Items {
		var excludedTagsAnnotation string
		var excludedTags []string
		cluster := &clusterList.Items[i]

		if cluster.Spec.Tags == nil {
			cluster.Spec.Tags = make(map[string]string)
		}

		excludedTagsAnnotation = cluster.Annotations["registry.ethos.adobe.com/excluded-tags"]

		if excludedTagsAnnotation != "" {
			excludedTags = strings.Split(excludedTagsAnnotation, ",")
		}

		// skip processing tags which are in excluded-tags list
		for key, value := range tag {
			if contains(key, excludedTags) {
				continue
			}
			cluster.Spec.Tags[key] = value
		}

		if err := s.Client.Update(context.TODO(), &clusterList.Items[i], &client.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// Retry function for updateClusterTags
func retry(f func(map[string]string) error, params map[string]string, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = f(params)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * 2)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func contains(item string, slice []string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
