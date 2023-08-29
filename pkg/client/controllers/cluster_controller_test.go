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

package controllers

import (
	"context"
	"time"

	registryv1 "github.com/adobe/cluster-registry/pkg/api/registry/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Client Controller", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	BeforeEach(func() {
		//
	})

	AfterEach(func() {
		//
	})

	Context("Controller annotations", func() {
		const (
			clusterName   = "test-cluster"
			namespaceName = "cluster-registry"
		)

		It("Should handle CA Cert", func() {
			ctx := context.Background()

			By("Creating cluster-registry namespace")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
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
					Namespace: namespaceName,
				},
				Spec: registryv1.ClusterSpec{
					Name:                   "cluster01-prod-useast1",
					ShortName:              "cluster01produseast1",
					APIServer:              registryv1.APIServer{Endpoint: "", CertificateAuthorityData: ""},
					Region:                 "useast1",
					CloudType:              "Azure",
					Environment:            "Prod",
					BusinessUnit:           "BU1",
					ChargebackBusinessUnit: "BU1",
					Offering:               []registryv1.Offering{},
					AccountID:              "",
					Tiers:                  []registryv1.Tier{},
					VirtualNetworks:        []registryv1.VirtualNetwork{},
					RegisteredAt:           "",
					Status:                 "Active",
					Phase:                  "Running",
					Type:                   "Shared",
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

			clusterLookupKey := types.NamespacedName{Name: clusterName, Namespace: namespaceName}
			createdCluster := &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, createdCluster)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			updatedCluster := &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				log.Log.Info(string(updatedCluster.Spec.APIServer.CertificateAuthorityData))
				if err != nil {
					return false
				}
				return updatedCluster.Spec.APIServer.CertificateAuthorityData == "_cert_data_"
			}, timeout, interval).Should(BeTrue())

			By("Adding annotation to the CRD to skip the CA Cert")
			cluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, cluster)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			cluster.Annotations = map[string]string{"registry.ethos.adobe.com/skip-ca-cert": "true"}
			cluster.Spec.APIServer.CertificateAuthorityData = "_custom_cert_data_"
			Expect(k8sClient.Update(ctx, cluster)).Should(Succeed())

			// give controller-runtime time to propagagte data into etcd
			time.Sleep(2 * time.Second)

			updatedCluster = &registryv1.Cluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, clusterLookupKey, updatedCluster)
				if err != nil {
					return false
				}
				return updatedCluster.Spec.APIServer.CertificateAuthorityData == "_custom_cert_data_"
			}, timeout, interval).Should(BeTrue())
		})
	})
})
