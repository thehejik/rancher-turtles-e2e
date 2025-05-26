export class ClusterManagementPage {
  visit() {
    cy.burgerMenuOperate('open');
    cy.contains('Cluster Management').click();
  }

  goToCAPIClusters() {
    cy.contains('CAPI').click();
    cy.contains('Clusters').click();
  }

  createNamespace(namespace: string) {
    cy.contains('Namespaces').click();
    cy.clickButton('Create');
    cy.typeValue('Name', namespace);
    cy.clickButton('Create');
    cy.contains(namespace).should('exist');
  }

  addFleetGitRepo(repoName: string, repoUrl: string, branch: string, path: string) {
    cy.contains('Continuous Delivery').click();
    cy.contains('Git Repos').click();
    cy.clickButton('Create');
    cy.typeValue('Name', repoName);
    cy.typeValue('Git Repo URL', repoUrl);
    cy.typeValue('Git Branch', branch);
    cy.typeValue('Path', path);
    cy.clickButton('Create');
    cy.contains(repoName).should('exist');
  }

  checkClusterStatus(clusterName: string, status: string, timeout = 300000) {
    cy.typeInFilter(clusterName);
    cy.contains(new RegExp(`${status}.*${clusterName}`), { timeout }).should('exist');
  }
}