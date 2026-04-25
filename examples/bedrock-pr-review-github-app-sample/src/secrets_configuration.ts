// ENGMODEL-OWNER-UNIT: FU-SECRETS-CONFIGURATION
// ENGMODEL-CODE-DESCRIPTION: loads GitHub and Bedrock secret configuration for review execution
import { SecretsManagerClient, GetSecretValueCommand } from "aws-sdk-go-v2/secretsmanager";

export class SecretsConfiguration {
  constructor(private readonly sm: SecretsManagerClient) {}

  // TRLC-LINKS: REQ-PRR-007
  async getGithubAppPrivateKey(secretId: string): Promise<string> {
    const output = await this.sm.send(new GetSecretValueCommand({ SecretId: secretId }));
    return output.SecretString ?? "";
  }
}
