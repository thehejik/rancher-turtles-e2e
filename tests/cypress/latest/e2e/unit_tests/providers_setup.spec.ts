/*
Copyright © 2022 - 2023 SUSE LLC
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

import '~/support/commands';
import * as cypressLib from '@rancher-ecp-qa/cypress-library';
import { qase } from 'cypress-qase-reporter/dist/mocha';

Cypress.config();
describe('Enable CAPI Providers', () => {

  const kubeadmProvider = 'kubeadm'
  const dockerProvider = 'docker'
  const amazonProvider = 'aws'
  const googleProvider = 'gcp'
  const azureProvider = 'azure'
  const fleetProvider = 'fleet'
  const fleetProviderVersion = 'v0.7.4'
  const vsphereProvider = 'vsphere'
  const vsphereProviderVersion = 'v1.12.0'
  const kubeadmProviderVersion = 'v1.9.5'
  const kubeadmBaseURL = 'https://github.com/kubernetes-sigs/cluster-api/releases/'
  const kubeadmProviderTypes = ['bootstrap', 'control plane']
  const providerNamespaces = ['capi-kubeadm-bootstrap-system', 'capi-kubeadm-control-plane-system', 'capd-system']
  const cloudProviderNamespaces = ['capi-kubeadm-bootstrap-system', 'capi-kubeadm-control-plane-system', 'capa-system'] //, 'capg-system', 'capz-system']
  const vsphereProviderNamespace = 'capv-system'

  beforeEach(() => {
    cy.login();
    cypressLib.burgerMenuToggle();
  });

  context('Local providers', { tags: '@short' }, () => {
    providerNamespaces.forEach(namespace => {
      it('Create CAPI Providers Namespaces - ' + namespace, () => {
        cy.createNamespace(namespace);
      })
    })

    kubeadmProviderTypes.forEach(providerType => {
      qase(27,
        it('Create Kubeadm Providers', () => {
          // Create CAPI Kubeadm providers
          if (providerType == 'control plane') {
            // https://github.com/kubernetes-sigs/cluster-api/releases/v1.9.5/control-plane-components.yaml
            const providerURL = kubeadmBaseURL + kubeadmProviderVersion + '/' + 'control-plane' + '-components.yaml'
            const providerName = kubeadmProvider + '-' + 'control-plane'
            const namespace = 'capi-kubeadm-control-plane-system'
            cy.addCustomProvider(providerName, namespace, kubeadmProvider, providerType, kubeadmProviderVersion, providerURL);
          } else {
            // https://github.com/kubernetes-sigs/cluster-api/releases/v1.9.5/bootstrap-components.yaml
            const providerURL = kubeadmBaseURL + kubeadmProviderVersion + '/' + providerType + '-components.yaml'
            const providerName = kubeadmProvider + '-' + providerType
            const namespace = 'capi-kubeadm-bootstrap-system'
            cy.addCustomProvider(providerName, namespace, kubeadmProvider, providerType, kubeadmProviderVersion, providerURL);
          }
        })
      );
    })

    qase(4,
      it('Create CAPD provider', () => {
        // Create Docker Infrastructure provider
        cy.addInfraProvider('Docker', dockerProvider, 'capd-system');
        var statusReady = 'Ready'
        statusReady = statusReady.concat(dockerProvider, 'infrastructure', dockerProvider, kubeadmProviderVersion)
        // TODO: add actual vs expected
        cy.contains(statusReady);
      })
    );

    it.skip('Custom Fleet addon config', () => {
      // Allows Fleet addon to be installed on specific clusters only

      const clusterName = 'local';
      const resourceKind = 'configMap';
      const resourceName = 'fleet-addon-config';
      const namespace = 'rancher-turtles-system';
      const patch = { data: { manifests: { isNestedIn: true, spec: { cluster: { selector: { matchLabels: { cni: 'by-fleet-addon-kindnet' } } } } } } };

      cy.patchYamlResource(clusterName, namespace, resourceKind, resourceName, patch);
    });

    it('Check Fleet addon provider', () => {
      // Fleet addon provider is provisioned automatically when enabled during installation
      cy.checkCAPIMenu();
      cy.contains('Providers').click();
      var statusReady = 'Ready'
      statusReady = statusReady.concat(fleetProvider, 'addon', fleetProvider, fleetProviderVersion);
      cy.contains(statusReady).scrollIntoView();
    });
  });

  context('vSphere provider', { tags: '@vsphere' }, () => {
    it('Create CAPI Providers Namespace - ' + vsphereProviderNamespace, () => {
      cy.createNamespace(vsphereProviderNamespace);
    })
    qase(40,
      it('Create CAPV provider', () => {
        // Create vsphere Infrastructure provider
        // See capv_rke2_cluster.spec.ts for more details about `vsphere_secrets_json_base64` structure
        const vsphere_secrets_json_base64 = Cypress.env("vsphere_secrets_json_base64")
        // Decode the base64 encoded secret and make json object
        const vsphere_secrets_json = JSON.parse(Buffer.from(vsphere_secrets_json_base64, 'base64').toString('utf-8'))
        // Access keys from the json object
        const vsphereUsername = vsphere_secrets_json.vsphere_username;
        const vspherePassword = vsphere_secrets_json.vsphere_password;
        const vsphereServer = vsphere_secrets_json.vsphere_server;
        const vspherePort = '443';
        cy.addCloudCredsVMware(vsphereProvider, vsphereUsername, vspherePassword, vsphereServer, vspherePort);
        cypressLib.burgerMenuToggle();
        cy.addInfraProvider('vsphere', vsphereProvider, vsphereProviderNamespace, vsphereProvider);
        var statusReady = 'Ready'
        statusReady = statusReady.concat(vsphereProvider, 'infrastructure', vsphereProvider, vsphereProviderVersion)
        cy.contains(statusReady);
      })
    );
  })

  context('Cloud Providers', { tags: '@full' }, () => {

    cloudProviderNamespaces.forEach(namespace => {
      it('Create CAPI Cloud Providers Namespaces - ' + namespace, () => {
        cy.createNamespace(namespace);
      })
    })

    kubeadmProviderTypes.forEach(providerType => {
      qase(27,
        it('Create Kubeadm Providers', () => {
          // Create CAPI Kubeadm providers
          if (providerType == 'control plane') {
            // https://github.com/kubernetes-sigs/cluster-api/releases/v1.9.5/control-plane-components.yaml
            const providerURL = kubeadmBaseURL + kubeadmProviderVersion + '/' + 'control-plane' + '-components.yaml'
            const providerName = kubeadmProvider + '-' + 'control-plane'
            const namespace = 'capi-kubeadm-control-plane-system'
            cy.addCustomProvider(providerName, namespace, kubeadmProvider, providerType, kubeadmProviderVersion, providerURL);
          } else {
            // https://github.com/kubernetes-sigs/cluster-api/releases/v1.9.5/bootstrap-components.yaml
            const providerURL = kubeadmBaseURL + kubeadmProviderVersion + '/' + providerType + '-components.yaml'
            const providerName = kubeadmProvider + '-' + providerType
            const namespace = 'capi-kubeadm-bootstrap-system'
            cy.addCustomProvider(providerName, namespace, kubeadmProvider, providerType, kubeadmProviderVersion, providerURL);
          }
        })
      );
    })

    qase(13,
      it('Create CAPA provider', () => {
        // Create AWS Infrastructure provider
        cy.addCloudCredsAWS(amazonProvider, Cypress.env('aws_access_key'), Cypress.env('aws_secret_key'));
        cypressLib.burgerMenuToggle();
        cy.addInfraProvider('Amazon', amazonProvider, 'capa-system', amazonProvider);
        var statusReady = 'Ready'
        statusReady = statusReady.concat(amazonProvider, 'infrastructure', amazonProvider, 'v2.8.1')
        cy.contains(statusReady);
      })
    );

    qase(28,
      it.skip('Create CAPG provider', () => {
        // Create GCP Infrastructure provider
        cy.addCloudCredsGCP(googleProvider, Cypress.env('gcp_credentials'));
        cypressLib.burgerMenuToggle();
        cy.addInfraProvider('Google', googleProvider, 'capg-system', googleProvider);
        var statusReady = 'Ready'
        statusReady = statusReady.concat(googleProvider, 'infrastructure', googleProvider, 'v1.9.0')
        cy.contains(statusReady, { timeout: 120000 });
      })
    );

    qase(20, it.skip('Create CAPZ provider', () => {
      // Create Azure Infrastructure provider
      cy.addCloudCredsAzure(azureProvider, Cypress.env('azure_client_id'), Cypress.env('azure_client_secret'), Cypress.env('azure_subscription_id'));
      cypressLib.burgerMenuToggle();
      cy.addInfraProvider('Azure', azureProvider, 'capz-system', azureProvider);
      var statusReady = 'Ready'
      statusReady = statusReady.concat(azureProvider, 'infrastructure', azureProvider)
      cy.contains(statusReady, { timeout: 180000 });
    })
    );
  })

});
