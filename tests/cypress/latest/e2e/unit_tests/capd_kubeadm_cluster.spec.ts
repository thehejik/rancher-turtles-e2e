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
describe('Import CAPD Kubeadm', { tags: '@short' }, () => {
  const timeout = 300000
  const classesRepo = 'classes'
  const clustersRepo = 'clusters'
  const clusterName = 'cluster1'
  const className = 'quick-start'
  const repoUrl = 'https://github.com/thehejik/rancher-turtles-e2e.git'
  const basePath = '/tests/assets/rancher-turtles-fleet-example/'
  const pathNames = ['clusterclass_autoimport']
  const branch = 'fleet-agent'

  beforeEach(() => {
    cy.login();
    cypressLib.burgerMenuToggle();
  });

  pathNames.forEach((path) => {
    it('Setup the namespace for importing', () => {
      if (path.includes('namespace_autoimport')) {
        cy.namespaceAutoImport('Enable');
      } else {
        cy.namespaceAutoImport('Disable');
      }
    })

    it('Add cluster fleet repo(s) - ' + path, () => {
      cypressLib.checkNavIcon('cluster-management').should('exist');
      var fullPath = basePath + path

      if (path.includes('clusterclass_autoimport')) {
        // Add cni gitrepo to fleet-default workspace
        // The cni gitrepo is scoped to quick-start class only by fleet.yaml
        cy.addFleetGitRepo('clusterclass-cni', repoUrl, branch, fullPath+'/cni', 'fleet-default');
        cypressLib.burgerMenuToggle();

        // Add classes fleet repo to fleel-local workspace
        fullPath = fullPath.concat('/', classesRepo)
        cy.addFleetGitRepo(classesRepo, repoUrl, branch, fullPath);
        fullPath = fullPath.replace(classesRepo, clustersRepo);
        cypressLib.burgerMenuToggle();
      }

      cy.addFleetGitRepo(clustersRepo, repoUrl, branch, fullPath);
    })

    if (path == 'namespace_autoimport') { var qase_id = 6 } else if (path == 'cluster_autoimport') { qase_id = 5 } else { qase_id = 0 }
    qase(qase_id,
      it('Auto import child CAPD cluster', () => {
        // Check child cluster is created and auto-imported
        cy.goToHome();
        cy.contains(new RegExp('Pending.*' + clusterName), { timeout: timeout });

        // Check cluster is Active
        cy.searchCluster(clusterName);
        cy.contains(new RegExp('Active.*' + clusterName), { timeout: timeout });
        // TODO: Check MachineSet unavailable status and use checkCAPIClusterActive
        cy.checkCAPIClusterProvisioned(clusterName);
      })
    );

    // TODO: Refactor for other paths
    if (path.includes('namespace_autoimport')) {
      qase(7,
        it.skip('Install App on imported cluster', { retries: 1 }, () => {
          // Click on imported CAPD cluster
          cy.contains(clusterName).click();
          // Install Chart
          cy.checkChart('Install', 'Monitoring', 'cattle-monitoring');
        })
      );

      qase(8,
        it.skip('Scale the imported CAPD cluster', () => {
          // Access CAPI cluster
          cy.checkCAPIMenu();
          cy.contains('Machine Deployments').click();
          cy.getBySel('sortable-table-0-action-button').click();
          cy.contains('Edit YAML')
            .click();
          cy.get('.CodeMirror')
            .then((editor) => {
              var text = editor[0].CodeMirror.getValue();
              text = text.replace(/replicas: 2/g, 'replicas: 3');
              editor[0].CodeMirror.setValue(text);
              cy.clickButton('Save');
          })

          // Check CAPI cluster status
          cy.contains('Machine Deployments').click();
          cy.get('.content > .count', { timeout: timeout }).should('have.text', '3');
          cy.checkCAPIClusterProvisioned(clusterName);
        })
      );
    }

    qase(9,
      it.skip('Remove imported CAPD cluster from Rancher Manager', { retries: 1 }, () => {
        // Check cluster is not deleted after removal
        cy.deleteCluster(clusterName);
        cy.goToHome();
        // kubectl get clusters.cluster.x-k8s.io
        // This is checked by ensuring the cluster is not available in navigation menu
        cy.contains(clusterName).should('not.exist');
        cy.checkCAPIClusterProvisioned(clusterName);
      })
    );

    qase(10,
      it.skip('Delete the CAPD cluster fleet repo(s) - ' + path, () => {
        if (path.includes('clusterclass_autoimport')) {
          // Remove the cni fleet repo from fleet-default workspace
          cy.removeFleetGitRepo('clusterclass-cni', true, 'fleet-default');
          // Remove the classes fleet repo
          cypressLib.burgerMenuToggle();
          cy.removeFleetGitRepo(classesRepo, true);
          // Remove the clusters fleet repo
          cypressLib.burgerMenuToggle();
          cy.removeFleetGitRepo(clustersRepo);
          
          // Wait until the following returns no clusters found
          cy.checkCAPIClusterDeleted(clusterName, timeout);
          // Remove the clusterclass
          cy.removeCAPIResource('Cluster Classes', className);
        } else {
          // Remove the clusters fleet repo
          cy.removeFleetGitRepo(clustersRepo);

          // Wait until the following returns no clusters found
          // This is checked by ensuring the cluster is not available in CAPI menu
          cy.checkCAPIClusterDeleted(clusterName, timeout);
        }

        // Ensure the cluster is not available in navigation menu
        cy.getBySel('side-menu').then(($menu) => {
          if ($menu.text().includes(clusterName)) {
            cy.deleteCluster(clusterName);
          }
        })
      })
    );

  })
});
