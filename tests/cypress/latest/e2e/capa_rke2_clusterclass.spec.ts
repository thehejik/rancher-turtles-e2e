import '~/support/commands';
import {qase} from 'cypress-qase-reporter/dist/mocha';
import {getClusterName, skipClusterDeletion, isRancherManagerVersion} from '~/support/utils';
import {capaResourcesCleanup, capiClusterDeletion, importedRancherClusterDeletion} from "~/support/cleanup_support";

Cypress.config();
describe('Import CAPA RKE2 Class-Cluster', { tags: '@full' }, () => {
  const timeout = 1200000
  const classNamePrefix = 'aws-rke2'
  const clusterName = getClusterName(classNamePrefix)
  const turtlesRepoUrl = 'https://github.com/rancher/turtles'
  const classesPath = 'examples/clusterclasses/aws/rke2'
  const clusterClassRepoName = 'aws-rke2-clusterclass'
  const providerName = 'aws'
  const accessKey = Cypress.env('aws_access_key')
  const secretKey = Cypress.env('aws_secret_key')

  beforeEach(() => {
    cy.login();
    cy.burgerMenuOperate('open');
  });

  it('Setup the namespace for importing', () => {
    cy.namespaceAutoImport('Disable');
  })

  // TODO: Create Provider via UI, ref: capi-ui-extension/issues/128
  it('Create AWS CAPIProvider & AWSClusterStaticIdentity', () => {
    cy.removeCAPIResource('Providers', providerName);
    cy.createCAPIProvider(providerName);
    cy.checkCAPIProvider(providerName);
    cy.createAWSClusterStaticIdentity(accessKey, secretKey);
  })

  qase(116,
    it('Add CAPA RKE2 ClusterClass Fleet Repo and check Applications', () => {
      cy.addFleetGitRepo(clusterClassRepoName, turtlesRepoUrl, 'main', classesPath, 'capi-classes')
      // Go to CAPI > ClusterClass to ensure the clusterclass is created
      cy.checkCAPIClusterClass(classNamePrefix);

      // Navigate to `local` cluster, More Resources > Fleet > Helm [O]|[Ap]ps and ensure the charts are active.
      cy.burgerMenuOperate('open');
      cy.contains('local').click();
      const helmPsMenuLocation = isRancherManagerVersion('2.12') ? ['More Resources', 'Fleet', 'Helm Ops'] : ['More Resources', 'Fleet', 'HelmApps'];
      cy.accesMenuSelection(helmPsMenuLocation);
      ['aws-ccm', 'aws-csi-driver'].forEach((app) => {
        cy.typeInFilter(app);
        cy.getBySel('sortable-cell-0-1').should('exist');
      })
    })
  );

  qase(110,
    it('Import CAPA RKE2 class-cluster using YAML', () => {
      cy.readFile('./fixtures/aws/capa-rke2-class-cluster.yaml').then((data) => {
        data = data.replace(/replace_cluster_name/g, clusterName)
        cy.importYAML(data, 'capi-clusters')
      });
      // Check CAPI cluster using its name
      cy.checkCAPICluster(clusterName);
    })
  );

  qase(111,
    it('Auto import child CAPA cluster', () => {
      // Go to Cluster Management > CAPI > Clusters and check if the cluster has provisioned
      cy.checkCAPIClusterProvisioned(clusterName, timeout);

      // Check child cluster is created and auto-imported
      // This is checked by ensuring the cluster is available in navigation menu
      cy.goToHome();
      cy.contains(clusterName).should('exist');

      // Check cluster is Active
      cy.searchCluster(clusterName);
      cy.contains(new RegExp('Active.*' + clusterName), { timeout: timeout });

      // Go to Cluster Management > CAPI > Clusters and check if the cluster has provisioned
      // Ensuring cluster is provisioned also ensures all the Cluster Management > Advanced > Machines for the given cluster are Active.
      cy.checkCAPIClusterActive(clusterName, timeout);
    })
  );

  qase(112,
    it('Install App on imported cluster', () => {
      // Click on imported CAPA cluster
      cy.contains(clusterName).click();

      // Install Chart
      // We install Logging chart instead of Monitoring, since this is relatively lightweight.
      cy.checkChart('Install', 'Logging', 'cattle-logging-system');
    })
  );

  it("Scale up imported CAPA cluster by patching class-cluster yaml", () => {
    cy.readFile('./fixtures/aws/capa-rke2-class-cluster.yaml').then((data) => {
      data = data.replace(/replicas: 2/g, 'replicas: 3')

      // workaround; these values need to be re-replaced before applying the scaling changes
      data = data.replace(/replace_cluster_name/g, clusterName)
      cy.importYAML(data, 'capi-clusters')
    })

    // Check CAPI cluster status
    cy.checkCAPIMenu();
    cy.contains('Machine Deployments').click();
    cy.typeInFilter(clusterName);
    cy.get('.content > .count', {timeout: timeout}).should('have.text', '3');
    cy.checkCAPIClusterActive(clusterName);
  })

  if (skipClusterDeletion) {
    qase(114,
      it('Remove imported CAPA cluster from Rancher Manager and Delete the CAPA cluster', { retries: 1 }, () => {
        // Delete the imported cluster
        // Ensure that the provisioned CAPI cluster still exists
        // this check can fail, ref: https://github.com/rancher/turtles/issues/1587
        importedRancherClusterDeletion(clusterName);
        // Remove CAPI Resources related to the cluster
        capiClusterDeletion(clusterName, timeout);
      })
    );

    qase(115,
      it('Delete the ClusterClass fleet repo and other resources', () => {
        // Remove the clusterclass repo
        cy.removeFleetGitRepo(clusterClassRepoName);
        // Cleanup other resources
        capaResourcesCleanup();
      })
    );
  }
});
