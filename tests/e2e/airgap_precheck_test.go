/*
Copyright © 2022 - 2026 SUSE LLC

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

package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// Global state for parsed versions
var (
	vTurtles      string
	vTurtlesChart string
	vCoreCAPI     string
	vFleet        string
	vAws          string
	vAzure        string
	vAso          string
	vGcp          string
	vVsphere      string
	vRke2         string
	vKubeadm      string
)

func fetchBytes(url string) []byte {
	resp, err := http.Get(url)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return body
}

func checkOCI(host, repo, tag string) {
	Expect(tag).ToNot(BeEmpty(), "Version for %s/%s is empty - cannot check OCI registry", host, repo)
	tag = strings.TrimSpace(tag) // Safety check for "unknown" or empty
	if tag == "" || tag == "unknown" {
		GinkgoWriter.Printf("⚠️ Skipping OCI check for %s/%s (Tag: %s)\n", host, repo, tag)
	}
	ref := fmt.Sprintf("%s/%s:%s", host, repo, tag)
	_, err := crane.Head(ref, crane.WithAuth(authn.Anonymous))
	Expect(err).ToNot(HaveOccurred(), "Artifact not found: %s", ref)
	GinkgoWriter.Printf("✅ Verified OCI: %s\n", ref)
}

func getAsoVersion() string {
	// ASO doesn't have a standard release pattern, so we may need to parse it differently
	asoURL := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/refs/tags/%s/charts/rancher-turtles-providers/values.yaml", vTurtles)
	var val struct {
		Images struct {
			InfrastructureAzure struct {
				AzureServiceOperator struct {
					Tag string `yaml:"tag"`
				} `yaml:"azureServiceOperator"`
			} `yaml:"infrastructureAzure"`
		} `yaml:"images"`
	}
	Expect(yaml.Unmarshal(fetchBytes(asoURL), &val)).To(Succeed())
	vAso = val.Images.InfrastructureAzure.AzureServiceOperator.Tag
	GinkgoWriter.Printf("ASO version from values.yaml: %s\n", vAso)
	return vAso
}

var _ = Describe("E2E - Airgap Precheck Tests", Label("airgap"), func() {
	BeforeEach(func() {
		// Test is suitable for Prime and rancher >= 2.14
		if !strings.Contains(rancherChannel, "prime") || isRancherManagerVersion("<2.14") {
			Skip(fmt.Sprintf("Skipping airgap precheck: requires prime channel and Rancher >= 2.14 (channel=%q, version=%s)", rancherChannel, rancherVersion))
		}
	})

	It("Phase 1: Data gathering", func() {
		By("Fetch and Parse Versions from Rancher & Turtles Sources", func() {
			// 1. Parse Rancher build.yaml
			buildURL := fmt.Sprintf("https://raw.githubusercontent.com/rancher/rancher/v%s/build.yaml", rancherVersion)
			var build struct {
				TurtlesVersion string `yaml:"turtlesVersion"`
			}
			Expect(yaml.Unmarshal(fetchBytes(buildURL), &build)).To(Succeed())
			GinkgoWriter.Printf("Turtles version from build.yaml: %s\n", build.TurtlesVersion)
			// Convert to OCI-compatible format (replace '+' with '_')
			vTurtlesChart = strings.ReplaceAll(build.TurtlesVersion, "+", "_")
			parts := strings.Split(build.TurtlesVersion, "+up")
			Expect(parts).To(HaveLen(2))
			vTurtles = "v" + strings.TrimPrefix(parts[1], "v")
			GinkgoWriter.Printf("Trimmed Turtles version: %s\n", vTurtles)

			// 2. Parse Turtles config-prime.yaml
			turtlesURL := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/refs/tags/%s/internal/controllers/clusterctl/config-prime.yaml", vTurtles)
			var configMap struct {
				Data struct {
					Clusterctl string `yaml:"clusterctl.yaml"`
				} `yaml:"data"`
			}
			Expect(yaml.Unmarshal(fetchBytes(turtlesURL), &configMap)).To(Succeed())

			var inner struct {
				Providers []struct {
					Name string `yaml:"name"`
					URL  string `yaml:"url"`
				} `yaml:"providers"`
			}
			Expect(yaml.Unmarshal([]byte(configMap.Data.Clusterctl), &inner)).To(Succeed())

			re := regexp.MustCompile(`\/releases\/([^/]+)`)
			for _, p := range inner.Providers {
				if p.URL == "" {
					continue
				}
				matches := re.FindStringSubmatch(p.URL)
				if len(matches) != 2 {
					continue
				}

				switch p.Name {
				case "cluster-api":
					vCoreCAPI = matches[1]
				case "rancher-fleet":
					vFleet = matches[1]
				case "aws":
					vAws = matches[1]
				case "azure":
					vAzure = matches[1]
				case "gcp":
					vGcp = matches[1]
				case "vsphere":
					vVsphere = matches[1]
				case "rke2":
					vRke2 = matches[1]
				case "kubeadm":
					vKubeadm = matches[1]
				}
			}

			// vAso is not listed in config-prime.yaml, will be fetched separately
			vAso = getAsoVersion()

		})
	})
	It("Phase 2: Validation", func() {
		var host string
		//if strings.Contains(rancherChannel, "prime") {
		// TODO make the registry invisible
		host = "stgregistry.suse.com"
		//}

		By("Verify all images exist in the OCI Registry", func() {
			// List of (repo_name, version_variable)
			images := []struct {
				repo   string
				verVar string
			}{
				{"rancher/turtles", vTurtles},
				{"rancher/cluster-api-controller", vCoreCAPI},
				{"rancher/cluster-api-addon-provider-fleet", vFleet},
				{"rancher/cluster-api-aws-controller", vAws},
				{"rancher/cluster-api-azure-controller", vAzure},
				{"rancher/cluster-api-gcp-controller", vGcp},
				{"rancher/cluster-api-vsphere-controller", vVsphere},
				{"rancher/cluster-api-provider-rke2-bootstrap", vRke2},
				{"rancher/cluster-api-provider-rke2-controlplane", vRke2},
				{"rancher/kubeadm-bootstrap-controller", vKubeadm},
				{"rancher/kubeadm-control-plane-controller", vKubeadm},
				{"rancher/charts/rancher-turtles-providers", vTurtlesChart},
				{"rancher/azureserviceoperator", vAso},
			}

			for _, i := range images {
				checkOCI(host, i.repo, i.verVar)
			}
		})

		By("Verify component manifest images exist in the registry", func() {
			// TODO make the registry invisible
			comp_host := "registry.suse.com" // Override or logic if needed
			components := []struct {
				repo   string
				verVar string
			}{
				{"rancher/cluster-api-controller-components", vCoreCAPI},
				{"rancher/cluster-api-addon-provider-fleet-components", vFleet},
				{"rancher/cluster-api-aws-controller-components", vAws},
				{"rancher/cluster-api-azure-controller-components", vAzure},
				{"rancher/cluster-api-gcp-controller-components", vGcp},
				{"rancher/cluster-api-provider-rke2-components", vRke2},
				{"rancher/cluster-api-vsphere-controller-components", vVsphere},
			}

			for _, c := range components {
				checkOCI(comp_host, c.repo, c.verVar)
			}
		})

		By("Verify images are listed in rancher-images.txt", func() {
			// TODO make the URL invisible
			url := fmt.Sprintf("https://github.com/rancher/rancher/releases/download/v%s2/rancher-images.txt", rancherVersion)
			if strings.Contains(rancherChannel, "prime") {
				url = fmt.Sprintf("https://prime.ribs.rancher.io/rancher/v%s/rancher-images.txt", rancherVersion)
			}
			content := string(fetchBytes(url))

			images := []struct {
				repo   string
				verVar string
			}{
				{"rancher/turtles", vTurtles},
				{"rancher/charts/rancher-turtles-providers", vTurtlesChart},
				{"rancher/cluster-api-controller", vCoreCAPI},
				{"rancher/cluster-api-addon-provider-fleet", vFleet},
				{"rancher/cluster-api-aws-controller", vAws},
				{"rancher/cluster-api-azure-controller", vAzure},
				{"rancher/cluster-api-gcp-controller", vGcp},
				{"rancher/cluster-api-vsphere-controller", vVsphere},
				{"rancher/cluster-api-provider-rke2-bootstrap", vRke2},
				{"rancher/cluster-api-provider-rke2-controlplane", vRke2},
				{"rancher/kubeadm-bootstrap-controller", vKubeadm},
				{"rancher/kubeadm-control-plane-controller", vKubeadm},
				{"rancher/azureserviceoperator", vAso},
			}

			for _, i := range images {
				expected := fmt.Sprintf("%s:%s", i.repo, i.verVar)
				Expect(content).To(ContainSubstring(expected), "Missing %s in rancher-images.txt", expected)
			}
		})

		By("Verify CAPI version match in package-env", func() {
			url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/rancher/refs/tags/v%s/scripts/package-env", rancherVersion)
			re := regexp.MustCompile(`CLUSTER_API_CONTROLLER_TAG=(v[0-9]+\.[0-9]+\.[0-9]+)`)
			matches := re.FindStringSubmatch(string(fetchBytes(url)))
			Expect(matches).To(HaveLen(2), "CLUSTER_API_CONTROLLER_TAG not found in package-env")
			GinkgoWriter.Printf("CAPI version in package-env: %s\n", matches[1])
			Expect(matches[1]).To(Equal(vCoreCAPI), "Mismatch between config-prime.yaml and package-env")
		})
	})
})
