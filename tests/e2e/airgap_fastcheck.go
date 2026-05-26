/// ###  airgap check whether images and components are available on registries for prime prime-rc and prime-alpha
/// ###
/// ### Initial check before we start to pull everything to AG env - this is fast check which will prevent
/// ### - FIrst get a list of provider versions in the given version / build - do not do that for community nor head builds - parse RANCHER_VERSION and strip channel/version where channel can be only prime[-rc|-alpha] and version contains two dots like 2.14.2[-alphaX|-rcY]
/// ### - from the used rancher version get know which turtles should be installed - use the version as a tag - look for turtlesVersion in https://github.com/rancher/rancher/blob/v2.14.2-alpha7/build.yaml and https://github.com/rancher/rancher/blob/v2.14.2-alpha7/scripts/package-env#L16 for CLUSTER_API_CONTROLLER_TAG (this is not needed, as the cluster-api is written but we need to be sure it matches with what is written in providers (see bellow)
/// ### - checkout turtles repo and use the version as a tag for the release turtlesVersion: 109.0.2+up0.26.2[-rc.3] - you have to strip what is after "up" - open like https://github.com/rancher/turtles/blob/v0.26.2-rc.3/internal/controllers/clusterctl/config-prime.yaml to get current provider versions list - store all provider names and their versions
/// ### - open rancher-images.txt and check whether the images are listed
/// ### - check stgregistry and/or registry and validate the images and components (manifests) are available with the given tag.

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var neededAirgapImageNames = []string{
	"rancher/cluster-api-addon-provider-fleet",
	"rancher/cluster-api-aws-controller",
	"rancher/cluster-api-azure-controller",
	"rancher/cluster-api-controller",
	"rancher/cluster-api-gcp-controller",
	"rancher/cluster-api-provider-rke2-bootstrap",
	"rancher/cluster-api-provider-rke2-controlplane",
	"rancher/cluster-api-vsphere-controller",
	"rancher/azureserviceoperator",
	"rancher/turtles",
	"rancher/kubeadm-bootstrap-controller",
	"rancher/kubeadm-control-plane-controller",
}

var neededAirgapComponentNames = []string{
	"rancher/cluster-api-aws-controller-components",
	"rancher/cluster-api-addon-provider-fleet-components",
	"rancher/cluster-api-azure-controller-components",
	"rancher/cluster-api-controller-components", // core, kubeadm and capd prividers
	"rancher/cluster-api-gcp-controller-components",
	"rancher/cluster-api-provider-metal3-components", // This is from outside turtles, we don't support that
	"rancher/cluster-api-provider-rke2-components",
	"rancher/cluster-api-vsphere-controller-components",
}

// Structural mappings for Rancher & Turtles YAML definitions
type RancherBuild struct {
	TurtlesVersion string `yaml:"turtlesVersion"`
}

type TurtlesConfigMap struct {
	Data struct {
		Clusterctl string `yaml:"clusterctl.yaml"`
	} `yaml:"data"`
}

type ClusterctlConfig struct {
	Providers []struct {
		Name string `yaml:"name"`
		URL  string `yaml:"url"`
	} `yaml:"providers"`
}

// Quick helper to fetch raw web text strings
func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func main() {
	rawVersion := strings.TrimSpace(os.Getenv("RANCHER_VERSION"))
	if rawVersion == "" {
		fmt.Println("❌ Missing RANCHER_VERSION environment variable")
		os.Exit(1)
	}

	// 1. Parse RANCHER_VERSION and strip optional prime channel prefix.
	// Supported examples:
	// - prime-rc/2.14.2-rc3
	// - prime-alpha/2.14.2-alpha7
	// - v2.14.2-rc3
	// - 2.14.2-alpha7
	re := regexp.MustCompile(`^(?:prime-(?:rc|alpha)[/:])?v?(2\.\d+\.\d+-(?:alpha|rc)\d+)$`)
	match := re.FindStringSubmatch(rawVersion)
	if len(match) < 2 {
		fmt.Printf("⏩ Skipping '%s': unsupported RANCHER_VERSION (expected prime-rc|prime-alpha with 2.x.y-(rc|alpha)N).\n", rawVersion)
		return
	}
	rancherVersion := "v" + strings.TrimPrefix(match[1], "v")
	fmt.Printf("🔍 Targeted Rancher Prime Release: %s\n", rancherVersion)

	// 2. Fetch Rancher build.yaml to find turtlesVersion tag
	buildURL := fmt.Sprintf("https://raw.githubusercontent.com/rancher/rancher/%s/build.yaml", rancherVersion)
	buildBytes, err := fetchURL(buildURL)
	if err != nil {
		fmt.Printf("❌ Failed to fetch build.yaml: %v\n", err)
		os.Exit(1)
	}

	var build RancherBuild
	if err := yaml.Unmarshal(buildBytes, &build); err != nil {
		fmt.Printf("❌ Failed to parse build.yaml: %v\n", err)
		os.Exit(1)
	}

	// 3. Normalize Turtles Tag (Strip everything before '+up')
	// e.g., "109.0.2+up0.26.2-rc.3" -> "v0.26.2-rc.3"
	parts := strings.Split(build.TurtlesVersion, "+up")
	if len(parts) < 2 {
		fmt.Printf("❌ Unexpected turtlesVersion structure: %s\n", build.TurtlesVersion)
		os.Exit(1)
	}
	turtlesTag := "v" + strings.TrimPrefix(parts[1], "v")
	fmt.Printf("🐢 Normalized Turtles Tag: %s\n", turtlesTag)

	// 4. Fetch Turtles config-prime.yaml
	turtlesURL := fmt.Sprintf("https://raw.githubusercontent.com/rancher/turtles/%s/internal/controllers/clusterctl/config-prime.yaml", turtlesTag)
	turtlesBytes, err := fetchURL(turtlesURL)
	if err != nil {
		fmt.Printf("❌ Failed to fetch config-prime.yaml: %v\n", err)
		os.Exit(1)
	}

	var cm TurtlesConfigMap
	if err := yaml.Unmarshal(turtlesBytes, &cm); err != nil {
		fmt.Printf("❌ Failed to parse config-prime ConfigMap: %v\n", err)
		os.Exit(1)
	}

	// 5. Extract inner provider definitions out of raw text block
	var clusterctl ClusterctlConfig
	if err := yaml.Unmarshal([]byte(cm.Data.Clusterctl), &clusterctl); err != nil {
		fmt.Printf("❌ Failed to parse inner clusterctl configurations: %v\n", err)
		os.Exit(1)
	}

	// 6. Match and display final provider tag matrix
	fmt.Println("\n📋 Detected Airgap Component Matrix:")
	versionRegex := regexp.MustCompile(`/releases/(v\d+\.\d+\.\d+[^/]*)/`)

	for _, provider := range clusterctl.Providers {
		verMatch := versionRegex.FindStringSubmatch(provider.URL)
		if len(verMatch) >= 2 {
			fmt.Printf("  • %-20s Tag: %s\n", provider.Name, verMatch[1])
			// Pro-tip: Here is where you could drop standard OCI library
			// elements to ping the registry directly if you expand the tool later.
		}
	}
}
