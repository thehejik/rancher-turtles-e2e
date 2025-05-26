import { ClusterManagementPage } from '~/support/pages/ClusterManagementPage';
import { testConfig } from '~/support/config/TestConfig';

describe('@pom Tests - All Variants', { tags: '@pom' }, () => {
  const clusterManagementPage = new ClusterManagementPage();

  beforeEach(() => {
    cy.login();
    clusterManagementPage.visit();
  });

  testConfig.variants.forEach((variant) => {
    describe(`Testing Variant: ${variant.name}`, () => {
      it('Create Namespace', () => {
        clusterManagementPage.createNamespace(variant.namespace);
      });

      it('Add Fleet Git Repository', () => {
        const { name, url, branch, path } = variant.repository;
        clusterManagementPage.addFleetGitRepo(name, url, branch, path);
      });

      it('Provision Cluster', () => {
        const clusterName = variant.cluster.generateName(); // Generate a unique cluster name
        clusterManagementPage.goToCAPIClusters();

        variant.cluster.expectedStatuses.forEach((status) => {
          clusterManagementPage.checkClusterStatus(clusterName, status);
        });
      });
    });
  });
});