/*
Copyright 2024 Adobe. All rights reserved.
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
	configv1 "github.com/adobe/cluster-registry/pkg/api/config/v1"
	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	monitoring "github.com/adobe/cluster-registry/pkg/monitoring/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"net/http"
	"net/http/httptest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

func newTestServer() *Server {
	metrics := monitoring.NewMetrics()
	metrics.Init(true)

	return &Server{
		Client:      k8sClient,
		Namespace:   "cluster-registry",
		BindAddress: "localhost:9999",
		Log:         ctrl.Log.WithName("webhook").WithName("Server"),
		Metrics:     metrics,
	}
}

func newTestAlert(name string, status string) *Alert {
	return &Alert{
		Receiver: "cluster-registry-webhook-testing",
		Status:   status,
		Alerts: []AlertItem{
			{
				Status: status,
				Labels: AlertLabels{
					Alertname: name,
				},
				StartsAt: time.Now(),
				EndsAt:   time.Now().Add(time.Minute),
			},
		},
		GroupLabels: GroupLabels{
			Alertname: name,
		},
		CommonLabels: CommonLabels{
			Alertname: name,
		},
	}
}

func newTestAlertAsJSON(name string, status string) string {
	alert := newTestAlert(name, status)
	b, err := json.Marshal(alert)
	if err != nil {
		return ""
	}
	return string(b)
}

var _ = Describe("Webhook Server", func() {
	var server *Server

	const (
		timeout  = time.Second * 60
		interval = time.Second
	)

	BeforeEach(func() {
		server = newTestServer()
	})

	AfterEach(func() {
		//
	})

	Context("Webhook Handler", func() {
		It("Should handle an empty/invalid request", func() {
			req := httptest.NewRequest(http.MethodGet, "/webhook", strings.NewReader(""))
			w := httptest.NewRecorder()
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("Should handle a valid request", func() {
			req := httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("CRCDeadMansSwitch", AlertStatusFiring)),
			)
			w := httptest.NewRecorder()
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		})

		It("Should handle DeadMansSwitch alert and record the metric", func() {
			// TODO: improve this test, at the moment it checks if the output has changed from the
			// initial state since we cannot reliable test for a timestamp value, and testutil does
			// not provide a regex/custom comparison

			By("Checking that the metric was initialized successfully")
			initial := strings.NewReader(`
			# HELP cluster_registry_cc_deadmansswitch_last_timestamp_seconds Last timestamp when a DeadMansSwitch alert was received.
			# TYPE cluster_registry_cc_deadmansswitch_last_timestamp_seconds gauge
			cluster_registry_cc_deadmansswitch_last_timestamp_seconds 0
			`)
			metric := server.Metrics.GetMetricByName("cluster_registry_cc_deadmansswitch_last_timestamp_seconds")
			Expect(metric).NotTo(BeNil())
			Expect(testutil.CollectAndCompare(metric, initial)).To(Succeed())

			By("Firing a DMS alert and expecting the metric have changed from its initial state")
			req := httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("CRCDeadMansSwitch", AlertStatusFiring)),
			)
			w := httptest.NewRecorder()
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
			Expect(testutil.CollectAndCompare(metric, initial)).ToNot(Succeed())
		})
	})

	Context("Alert Mapping", func() {

		const (
			clusterName = "test-cluster"
		)

		It("Should handle an alert based on existing alert mapping", func() {
			ctx := context.Background()

			server.AlertMap = []configv1.AlertRule{
				{
					AlertName: "TestAlertMapping",
					OnFiring: map[string]string{
						"my-tag": "on",
					},
					OnResolved: map[string]string{
						"my-tag": "off",
					},
				},
			}

			By("Creating cluster-registry namespace")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: server.Namespace,
				},
			}
			Expect(k8sClient.Create(ctx, namespace)).Should(Succeed())

			By("Creating a new Cluster CRD")
			cluster := &registryv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Cluster",
					APIVersion: registryv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: server.Namespace,
				},
				Spec: registryv1.ClusterSpec{
					Name:                   "cluster01-prod-useast1",
					ShortName:              "cluster01produseast1",
					APIServer:              registryv1.APIServer{Endpoint: "", CertificateAuthorityData: ""},
					ArgoInstance:           "argocd-prod-gen-01.cluster01-prod-useast1.example.com",
					Region:                 "useast1",
					CloudType:              "Azure",
					Environment:            "Prod",
					BusinessUnit:           "BU1",
					ChargebackBusinessUnit: "BU1",
					ChargedBack:            ptr.To(true),
					Offering:               []registryv1.Offering{},
					AccountID:              "",
					Tiers:                  []registryv1.Tier{},
					VirtualNetworks:        []registryv1.VirtualNetwork{},
					RegisteredAt:           "",
					Status:                 "Active",
					Phase:                  "Running",
					Type:                   "Shared",
					MaintenanceGroup:       "B",
					Extra: registryv1.Extra{
						DomainName:       "",
						LbEndpoints:      map[string]string{},
						LoggingEndpoints: []map[string]string{},
						EcrIamArns:       map[string][]string{},
						EgressPorts:      "",
						NFSInfo:          []map[string]string{},
					},
					AllowedOnboardingTeams: []registryv1.AllowedOnboardingTeam{},
					Capabilities:           []string{},
					PeerVirtualNetworks:    []registryv1.PeerVirtualNetwork{},
					LastUpdated:            "",
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			clusterLookupKey := types.NamespacedName{Name: clusterName, Namespace: server.Namespace}
			createdCluster := &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, createdCluster)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Wait for the post creation updates from the ClusterReconciler
			updatedCluster := &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Firing an alert and having it be mapped correctly")
			req := httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("TestAlertMapping", AlertStatusFiring)),
			)
			w := httptest.NewRecorder()
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))

			updatedCluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				if err != nil {
					return false
				}
				return updatedCluster.Spec.Tags["my-tag"] == "on"
			}, timeout, interval).Should(BeTrue())

			By("Resolving the alert and having it be mapped correctly")
			req = httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("TestAlertMapping", AlertStatusResolved)),
			)
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))

			updatedCluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				if err != nil {
					return false
				}
				return updatedCluster.Spec.Tags["my-tag"] == "off"
			}, timeout, interval).Should(BeTrue())

			By("Adding annotation to the CRD to ignore specific tag")
			cluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, cluster)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			temp := cluster.DeepCopy()
			temp.SetAnnotations(map[string]string{"registry.ethos.adobe.com/excluded-tags": "other-tag,my-tag"})
			Eventually(func() bool {
				err := k8sClient.Patch(ctx, temp, client.MergeFrom(cluster))
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// give controller-runtime time to propagagte data into etcd
			time.Sleep(2 * time.Second)

			By("Firing an alert and check if it is ignored")
			req = httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("TestAlertMapping", AlertStatusFiring)),
			)
			w = httptest.NewRecorder()
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))

			updatedCluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				if err != nil {
					return false
				}
				return updatedCluster.Spec.Tags["my-tag"] == "off"
			}, timeout, interval).Should(BeTrue())

			By("Resolving the alert and check if it is ignored")
			req = httptest.NewRequest(
				http.MethodGet,
				"/webhook",
				strings.NewReader(newTestAlertAsJSON("TestAlertMapping", AlertStatusResolved)),
			)
			server.webhookHandler(w, req)
			Expect(w.Result().StatusCode).To(Equal(http.StatusOK))

			updatedCluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				if err != nil {
					return false
				}
				fmt.Println(updatedCluster.ObjectMeta.ResourceVersion)
				return updatedCluster.Spec.Tags["my-tag"] == "off"
			}, timeout, interval).Should(BeTrue())
		})
	})
})
