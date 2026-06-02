package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
)

func init() {
	// To suppress crane's default logging which can be quite verbose during registry checks, we redirect it to Discard.
	logs.Debug.SetOutput(io.Discard)
  	logs.Warn.SetOutput(io.Discard)
  	logs.Progress.SetOutput(io.Discard)
}

// Global Struct Definition
type AirgapComponent struct {
	Name    string
	Type    string
	Version string
}

// State variables shared across It blocks
var (
	turtlesVersion   string
	versionMap       = make(map[string]string)
	airgapCollection []AirgapComponent
)

// Shared static lists
var neededAirgapImageNames = []string{
	"rancher/cluster-api-addon-provider-fleet",
	"rancher/cluster-api-aws-controller",
	"rancher/cluster-api-azure-controller",
	"rancher/cluster-api-controller",
	"rancher/cluster-api-gcp-controller",
	"rancher/cluster-api-provider-rke2-bootstrap",
	"rancher/cluster-api-provider-rke2-controlplane",
	"rancher/cluster-api-vsphere-controller",
	"rancher/azureserviceoperator", // TODO the version of ASO is not defined anywhere but in azure components file
	"rancher/turtles",
	"rancher/kubeadm-bootstrap-controller",
	"rancher/kubeadm-control-plane-controller",
	// docker image is not released within rancher org
	"rancher/charts/rancher-turtles-providers", // This helm chart is concidered as a image
}

var neededAirgapComponentNames = []string{
	"rancher/cluster-api-aws-controller-components",
	"rancher/cluster-api-addon-provider-fleet-components",
	"rancher/cluster-api-azure-controller-components",
	"rancher/cluster-api-controller-components", // contains manifests for core cluster-api, kubeadm and capd providers
	"rancher/cluster-api-gcp-controller-components",
	"rancher/cluster-api-provider-rke2-components",
	"rancher/cluster-api-vsphere-controller-components",
}

// Simple HTTP fetcher helper
func fetchRaw(url string) []byte {
	resp, err := http.Get(url)
	Expect(err).ToNot(HaveOccurred(), "Failed to download %s", url)
	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(http.StatusOK), "Non-200 status for %s", url)
	body, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	return body
}

var _ = Describe("E2E - Airgap Precheck Tests", Label("airgap"), func() {

	It("Step 1: Check if RANCHER_VERSION is a prime release", func() {
		Expect(rancherChannel).To(ContainSubstring("prime"), "RANCHER_VERSION does not indicate a prime release (missing 'prime' prefix) - skipping airgap precheck")
	})

	It("Step 2: Get Turtles version defined in build.yaml for given rancher release", func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/rancher/v%s/build.yaml", rancherVersion)
		body := fetchRaw(url)

		var build struct {
			TurtlesVersion string `yaml:"turtlesVersion"`
		}
		Expect(yaml.Unmarshal(body, &build)).To(Succeed())
		Expect(build.TurtlesVersion).ToNot(BeEmpty(), "turtlesVersion missing in build.yaml")

		// Normalize version (e.g., 109.0.2+up0.26.2 -> v0.26.2)
		turtlesChartVersion := build.TurtlesVersion
		versionMap["rancher-turtles-providers"] = turtlesChartVersion // the version should be the same as the turtles chart version
		parts := strings.Split(build.TurtlesVersion, "+up")
		Expect(parts).To(HaveLen(2), "Unexpected turtlesVersion format (missing '+up' separator)")
		turtlesVersion = "v" + strings.TrimPrefix(parts[1], "v")
		versionMap["turtles"] = turtlesVersion
		GinkgoWriter.Printf("🐢 Normalized Turtles Version: %s\n", turtlesVersion)
	})

	It("Step 3: Parse config-prime.yaml and extract clusterctl providers", func() {
		url := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/refs/tags/%s/internal/controllers/clusterctl/config-prime.yaml", turtlesVersion)
		body := fetchRaw(url)

		var configMap struct {
			Data struct {
				Clusterctl string `yaml:"clusterctl.yaml"`
			} `yaml:"data"`
		}
		Expect(yaml.Unmarshal(body, &configMap)).To(Succeed())

		var inner struct {
			Providers []struct {
				Name string `yaml:"name"`
				URL  string `yaml:"url"`
				Type string `yaml:"type"`
			} `yaml:"providers"`
		}
		Expect(yaml.Unmarshal([]byte(configMap.Data.Clusterctl), &inner)).To(Succeed())

		// Extract version regex match and store in lookup map
		re := regexp.MustCompile(`\/releases\/([^/]+)`)
		for _, p := range inner.Providers {
			if p.URL == "" {
				continue
			}
			matches := re.FindStringSubmatch(p.URL)
			if len(matches) == 2 {
				versionMap[p.Name] = matches[1]

				// Add base providers directly to our final target collection
				airgapCollection = append(airgapCollection, AirgapComponent{
					Name:    p.Name,
					Type:    p.Type,
					Version: matches[1],
				})
			}
		}
	})

	It("Step 4: Resolve and map required container images", func() {
		for _, img := range neededAirgapImageNames {
			version := "unknown"

			switch {
			case img == "rancher/turtles":
				version = versionMap["turtles"]

			case img == "rancher/charts/rancher-turtles-providers":
				// This is actually a helm chart but can be concidered as a image for the later checks
				// We need to sanitze the version string to match the actual OCI tag format used in the registry, which replaces "+" with "_"
				version = strings.ReplaceAll(versionMap["rancher-turtles-providers"], "+", "_")

			case img == "rancher/cluster-api-controller":
				version = versionMap["cluster-api"]

			case img == "rancher/azureserviceoperator":
				// TODO Inherit Azure infrastructure version context directly is not the way
				// version = versionMap["azure"]

			case strings.Contains(img, "cluster-api-addon-provider-fleet"):
				version = versionMap["rancher-fleet"]

			default:
				// Exact keyword lookup for traditional provider components
				for _, provider := range []string{"aws", "azure", "gcp", "vsphere", "rke2", "kubeadm"} {
					if strings.Contains(img, provider) {
						version = versionMap[provider]
						break
					}
				}
			}

			airgapCollection = append(airgapCollection, AirgapComponent{
				Name:    img,
				Type:    "ContainerImage",
				Version: version,
			})
		}
	})

	It("Step 5: Resolve and map required component manifests", func() {
		for _, comp := range neededAirgapComponentNames {
			version := "unknown"

			switch {
			case comp == "rancher/cluster-api-controller-components":
				version = versionMap["cluster-api"]

			case strings.Contains(comp, "cluster-api-addon-provider-fleet-components"):
				version = versionMap["rancher-fleet"]

			default:
				// Exact keyword lookup for remaining provider files
				for _, provider := range []string{"aws", "azure", "gcp", "vsphere", "rke2", "kubeadm"} {
					if strings.Contains(comp, provider) {
						version = versionMap[provider]
						break
					}
				}
			}

			airgapCollection = append(airgapCollection, AirgapComponent{
				Name:    comp,
				Type:    "ComponentManifest",
				Version: version,
			})
		}
	})

	It("Step 6: Output and verify final collection summary", func() {
		Expect(airgapCollection).ToNot(BeEmpty(), "No assets processed into the airgap collection")

		GinkgoWriter.Printf("\n📦 Summary: Collected %d Total Airgap Artifacts\n", len(airgapCollection))
		for _, item := range airgapCollection {
			GinkgoWriter.Printf("  [%s] %s -> %s\n", item.Type, item.Name, item.Version)
		}
	})

   It("Step 7: Verify all required artifacts exist on registry", func() {
		for _, item := range airgapCollection {
			registryHost := "stgregistry.suse.com" // TODO load this from env/secret
			if item.Type != "ContainerImage" && item.Type != "ComponentManifest" {
				continue
			}

			if item.Version == "unknown" || item.Version == "" {
				GinkgoWriter.Printf("⚠️  Skipping validation for %s (%s) due to unknown version\n", item.Name, item.Type)
				continue
			}

			if item.Type == "ComponentManifest" {
				registryHost = "registry.suse.com" // TODO use secret - components are always stored on registry.suse.com
			}

			// Clean repository name to build a fully qualified image reference
			repoName := strings.TrimPrefix(item.Name, registryHost+"/")
			fullyQualifiedRef := fmt.Sprintf("%s/%s:%s", registryHost, repoName, item.Version)

			// crane.Head automatically intercepts the 401, fetches the anonymous bearer token,
			// appends the standard OCI Accept headers, and performs the lightweight HEAD check.
			_, err := crane.Head(fullyQualifiedRef, crane.WithAuth(authn.Anonymous))

			// Assert existence
			Expect(err).ToNot(HaveOccurred(), "OCI Artifact [%s] -> %s:%s NOT found on registry", item.Type, repoName, item.Version)

			//GinkgoWriter.Printf("✅ Verified OCI %s existence: %s\n", item.Type, fullyQualifiedRef)
			GinkgoWriter.Printf("✅ Verified OCI %s existence: %s:%s\n", item.Type, repoName, item.Version)
		}
	})

	It("Step 8: Verify that all images are listed in rancher-images.txt", func() {
		url := fmt.Sprintf("https://prime.ribs.rancher.io/rancher/v%s/rancher-images.txt", rancherVersion) // TODO USE SECRET HERE
		body := fetchRaw(url)
		imagesList := strings.Split(string(body), "\n")
		for _, item := range airgapCollection {
			if item.Type != "ContainerImage" || item.Version == "unknown" {
				continue
			}
			imageRef := fmt.Sprintf("rancher/%s:%s", strings.TrimPrefix(item.Name, "rancher/"), item.Version)
			Expect(imagesList).To(ContainElement(imageRef), "Image %s is missing from rancher-images.txt", imageRef)
			GinkgoWriter.Printf("✅ Verified image listed in rancher-images.txt: %s\n", imageRef)
		}
	})

	It("Step 9: Verify that CLUSTER_API_CONTROLLER_TAG is set to correct value", func() {
		//TODO
	})

})
