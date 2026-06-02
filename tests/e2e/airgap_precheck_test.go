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

// Global variables holding ONLY raw versions discovered in the release
var (
	vTurtles      string // e.g., v0.26.2
	vTurtlesChart string // e.g., 109.0.2_up0.26.2
	vCoreCAPI     string // e.g., v1.12.7
	vFleet        string // e.g., v0.14.1
	vAws          string // e.g., v2.11.1
	vAzure        string // e.g., v1.22.0
	vAso          string // e.g., v1.15.0 // TODO get it from https://github.com/rancher/turtles/blob/v0.26.2/charts/rancher-turtles-providers/values.yaml
	vGcp          string // e.g., v1.11.1
	vVsphere      string // e.g., v1.15.2
	vRke2         string // e.g., v0.24.4
	vKubeadm      string // e.g., v1.12.7
)

// Simple HTTP helper to stop repeating error handling
func fetchBytes(url string) []byte {
	resp, err := http.Get(url)
	Expect(err).ToNot(HaveOccurred())
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return body
}

// Registry checker helper to stop repeating crane calls
func checkOCI(host, repo, tag string) {
	Expect(tag).ToNot(BeEmpty(), "Version for %s/%s is empty - cannot check OCI registry", host, repo)
	ref := fmt.Sprintf("%s/%s:%s", host, repo, tag)
	_, err := crane.Head(ref, crane.WithAuth(authn.Anonymous))
	Expect(err).ToNot(HaveOccurred(), "Artifact not found: %s", ref)
	GinkgoWriter.Printf("✅ Verified OCI: %s\n", ref)
}

var _ = Describe("E2E - Airgap Precheck Tests", Label("airgap"), func() {

	It("Step 1: Check Release Channel", func() {
		Expect(rancherChannel).To(ContainSubstring("prime"), "Skipping test: channel is not prime")
	})

	It("Step 2: Parse Turtles Versions from Rancher build.yaml", func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/rancher/v%s/build.yaml", rancherVersion)

		var build struct {
			TurtlesVersion string `yaml:"turtlesVersion"`
		}
		Expect(yaml.Unmarshal(fetchBytes(url), &build)).To(Succeed())

		vTurtlesChart = strings.ReplaceAll(build.TurtlesVersion, "+", "_")

		parts := strings.Split(build.TurtlesVersion, "+up")
		Expect(parts).To(HaveLen(2))
		vTurtles = "v" + strings.TrimPrefix(parts[1], "v")
	})

	It("Step 3: Parse Provider Versions from Turtles config-prime.yaml", func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/refs/tags/%s/internal/controllers/clusterctl/config-prime.yaml", vTurtles)

		var configMap struct {
			Data struct {
				Clusterctl string `yaml:"clusterctl.yaml"`
			} `yaml:"data"`
		}
		Expect(yaml.Unmarshal(fetchBytes(url), &configMap)).To(Succeed())

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

			// Directly map specific variables without loops
			switch p.Name {
			case "cluster-api":
				vCoreCAPI = matches[1]
			case "rancher-fleet":
				vFleet = matches[1]
			case "aws":
				vAws = matches[1]
			case "azure":
				vAzure = matches[1]
			// aso not present
			case "aso":
				vAso = "unknown" // matches[1] // Note: ASO doesn't follow the same naming convention, so we use "aso" as the case and map it to vAso
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
	})

	It("Step 4: Verify Container Images on Registry", func() {
		h := ""
		if strings.Contains(rancherChannel, "prime") {
			h = "stgregistry.suse.com"
		}

		checkOCI(h, "rancher/turtles", vTurtles)
		checkOCI(h, "rancher/cluster-api-controller", vCoreCAPI)
		checkOCI(h, "rancher/cluster-api-addon-provider-fleet", vFleet)
		checkOCI(h, "rancher/cluster-api-aws-controller", vAws)
		checkOCI(h, "rancher/cluster-api-azure-controller", vAzure)
		checkOCI(h, "rancher/cluster-api-gcp-controller", vGcp)
		checkOCI(h, "rancher/cluster-api-vsphere-controller", vVsphere)
		checkOCI(h, "rancher/cluster-api-provider-rke2-bootstrap", vRke2)
		checkOCI(h, "rancher/cluster-api-provider-rke2-controlplane", vRke2)
		checkOCI(h, "rancher/kubeadm-bootstrap-controller", vKubeadm)
		checkOCI(h, "rancher/kubeadm-control-plane-controller", vKubeadm)
		checkOCI(h, "rancher/azureserviceoperator", vAso)
	})

	It("Step 5: Verify Component Manifests", func() {
		h := "registry.suse.com"

		checkOCI(h, "rancher/cluster-api-controller-components", vCoreCAPI)
		checkOCI(h, "rancher/charts/rancher-turtles-providers", vTurtlesChart)
		checkOCI(h, "rancher/cluster-api-addon-provider-fleet-components", vFleet)
		checkOCI(h, "rancher/cluster-api-aws-controller-components", vAws)
		checkOCI(h, "rancher/cluster-api-azure-controller-components", vAzure)
		checkOCI(h, "rancher/cluster-api-gcp-controller-components", vGcp)
		checkOCI(h, "rancher/cluster-api-provider-rke2-components", vRke2)
		checkOCI(h, "rancher/cluster-api-vsphere-controller-components", vVsphere)
	})

	It("Step 6: Verify Images are in rancher-images.txt", func() {
		url := fmt.Sprintf("https://github.com/rancher/rancher/releases/download/v%s2/rancher-images.txt", rancherVersion)
		if strings.Contains(rancherChannel, "prime") {
			url = fmt.Sprintf("https://prime.ribs.rancher.io/rancher/v%s/rancher-images.txt", rancherVersion)
		}
		txt := string(fetchBytes(url))

		// Construct and check direct strings one by one
		Expect(txt).To(ContainSubstring(fmt.Sprintf("rancher/turtles:%s", vTurtles)))
		Expect(txt).To(ContainSubstring(fmt.Sprintf("rancher/charts/rancher-turtles-providers:%s", vTurtlesChart)))
		Expect(txt).To(ContainSubstring(fmt.Sprintf("rancher/cluster-api-controller:%s", vCoreCAPI)))
		// ... repeat for the rest of your required image tags if desired
	})
	It("Step 7: Verify ASO version is documented in Turtles values.yaml", func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/refs/tags/%s/charts/rancher-turtles-providers/values.yaml", vTurtles)

		var values struct {
			Images struct {
				InfrastructureAzure struct {
					AzureServiceOperator struct {
						Tag string `yaml:"tag"`
					} `yaml:"azureServiceOperator"`
				} `yaml:"infrastructureAzure"`
			} `yaml:"images"`
		}
		Expect(yaml.Unmarshal(fetchBytes(url), &values)).To(Succeed())
		vAso = values.Images.InfrastructureAzure.AzureServiceOperator.Tag
		Expect(vAso).ToNot(BeEmpty(), "ASO tag in values.yaml is empty")
	})
})
