import testConfigJson from '../../config/testConfig.json';

export interface RepositoryConfig {
  name: string;
  url: string;
  branch: string;
  path: string;
}

export interface ClusterConfig {
  baseName: string;
  expectedStatuses: string[];
  generateName(): string;
}

export interface VariantConfig {
  name: string;
  namespace: string;
  repository: RepositoryConfig;
  cluster: ClusterConfig;
}

export class TestConfig {
  variants: VariantConfig[];

  constructor(config: any) {
    this.variants = config.variants.map((variant: any) => ({
      ...variant,
      cluster: {
        ...variant.cluster,
        generateName: function () {
          return `${this.baseName}-${Date.now()}`;
        }
      }
    }));
  }

  getVariantByName(name: string): VariantConfig | undefined {
    return this.variants.find((variant) => variant.name === name);
  }
}

// Create an instance of TestConfig
export const testConfig = new TestConfig(testConfigJson);