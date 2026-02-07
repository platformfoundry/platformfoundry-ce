# Tutorial: CI/CD Integration

Integrate PlatformFoundry with your CI/CD pipelines for automated platform deployments.

## Overview

This tutorial covers:

- GitHub Actions integration
- GitLab CI integration
- Jenkins pipeline
- Automated testing and validation

## GitHub Actions

### Basic Workflow

Create `.github/workflows/platform.yml`:

```yaml
name: Platform CI/CD

on:
  push:
    branches: [main]
    paths:
      - 'platform/**'
      - 'infrastructure/**'
  pull_request:
    branches: [main]

env:
  PF_VERSION: "0.1.0"

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install PlatformFoundry
        run: |
          curl -sSL https://github.com/platformfoundry/pf-ce/releases/download/v${{ env.PF_VERSION }}/pf-linux-amd64 -o pf
          chmod +x pf
          sudo mv pf /usr/local/bin/

      - name: Validate Configuration
        run: pf validate -f platform/platform.yaml

      - name: Lint Policies
        run: pf policy test --all

  plan:
    needs: validate
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v4

      - name: Install PlatformFoundry
        run: |
          curl -sSL https://github.com/platformfoundry/pf-ce/releases/download/v${{ env.PF_VERSION }}/pf-linux-amd64 -o pf
          chmod +x pf
          sudo mv pf /usr/local/bin/

      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1

      - name: Generate Plan
        id: plan
        run: |
          pf plan -f platform/platform.yaml -o plan.txt
          echo "plan<<EOF" >> $GITHUB_OUTPUT
          cat plan.txt >> $GITHUB_OUTPUT
          echo "EOF" >> $GITHUB_OUTPUT

      - name: Comment Plan on PR
        uses: actions/github-script@v7
        with:
          script: |
            const plan = `${{ steps.plan.outputs.plan }}`;
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## Platform Plan\n\`\`\`\n${plan}\n\`\`\``
            });

  deploy:
    needs: validate
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    environment: production
    steps:
      - uses: actions/checkout@v4

      - name: Install PlatformFoundry
        run: |
          curl -sSL https://github.com/platformfoundry/pf-ce/releases/download/v${{ env.PF_VERSION }}/pf-linux-amd64 -o pf
          chmod +x pf
          sudo mv pf /usr/local/bin/

      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1

      - name: Deploy Platform
        run: pf apply -f platform/platform.yaml --auto-approve

      - name: Verify Deployment
        run: |
          pf get platforms
          pf health check --platform=production
```

### Multi-Environment Workflow

```yaml
name: Multi-Environment Deploy

on:
  push:
    branches:
      - main
      - 'release/*'

jobs:
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4

      - name: Deploy to Staging
        run: |
          pf apply -f platform/platform.yaml \
            --environment=staging \
            --auto-approve

      - name: Run Integration Tests
        run: pf test integration --environment=staging

  deploy-production:
    needs: deploy-staging
    runs-on: ubuntu-latest
    environment: production
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4

      - name: Deploy to Production
        run: |
          pf apply -f platform/platform.yaml \
            --environment=production \
            --auto-approve
```

## GitLab CI

Create `.gitlab-ci.yml`:

```yaml
stages:
  - validate
  - plan
  - deploy

variables:
  PF_VERSION: "0.1.0"

.pf-setup: &pf-setup
  before_script:
    - curl -sSL https://github.com/platformfoundry/pf-ce/releases/download/v${PF_VERSION}/pf-linux-amd64 -o pf
    - chmod +x pf
    - mv pf /usr/local/bin/

validate:
  stage: validate
  <<: *pf-setup
  script:
    - pf validate -f platform/platform.yaml
    - pf policy test --all
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH

plan:
  stage: plan
  <<: *pf-setup
  script:
    - pf plan -f platform/platform.yaml -o plan.txt
    - cat plan.txt
  artifacts:
    paths:
      - plan.txt
    expire_in: 1 week
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"

deploy-staging:
  stage: deploy
  <<: *pf-setup
  environment:
    name: staging
    url: https://staging.example.com
  script:
    - pf apply -f platform/platform.yaml --environment=staging --auto-approve
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH

deploy-production:
  stage: deploy
  <<: *pf-setup
  environment:
    name: production
    url: https://example.com
  script:
    - pf apply -f platform/platform.yaml --environment=production --auto-approve
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
      when: manual
  needs:
    - deploy-staging
```

## Jenkins Pipeline

Create `Jenkinsfile`:

```groovy
pipeline {
    agent any

    environment {
        PF_VERSION = '0.1.0'
        AWS_CREDENTIALS = credentials('aws-credentials')
    }

    stages {
        stage('Setup') {
            steps {
                sh '''
                    curl -sSL https://github.com/platformfoundry/pf-ce/releases/download/v${PF_VERSION}/pf-linux-amd64 -o pf
                    chmod +x pf
                    sudo mv pf /usr/local/bin/
                '''
            }
        }

        stage('Validate') {
            steps {
                sh 'pf validate -f platform/platform.yaml'
                sh 'pf policy test --all'
            }
        }

        stage('Plan') {
            when {
                changeRequest()
            }
            steps {
                withCredentials([[$class: 'AmazonWebServicesCredentialsBinding',
                                  credentialsId: 'aws-credentials']]) {
                    sh 'pf plan -f platform/platform.yaml -o plan.txt'
                }
                archiveArtifacts artifacts: 'plan.txt'
            }
        }

        stage('Deploy Staging') {
            when {
                branch 'main'
            }
            steps {
                withCredentials([[$class: 'AmazonWebServicesCredentialsBinding',
                                  credentialsId: 'aws-credentials']]) {
                    sh 'pf apply -f platform/platform.yaml --environment=staging --auto-approve'
                }
            }
        }

        stage('Integration Tests') {
            when {
                branch 'main'
            }
            steps {
                sh 'pf test integration --environment=staging'
            }
        }

        stage('Deploy Production') {
            when {
                branch 'main'
            }
            input {
                message "Deploy to production?"
                ok "Deploy"
            }
            steps {
                withCredentials([[$class: 'AmazonWebServicesCredentialsBinding',
                                  credentialsId: 'aws-credentials']]) {
                    sh 'pf apply -f platform/platform.yaml --environment=production --auto-approve'
                }
            }
        }
    }

    post {
        always {
            cleanWs()
        }
        failure {
            slackSend channel: '#platform-alerts',
                      color: 'danger',
                      message: "Platform deployment failed: ${env.BUILD_URL}"
        }
        success {
            slackSend channel: '#platform-deploys',
                      color: 'good',
                      message: "Platform deployed successfully: ${env.BUILD_URL}"
        }
    }
}
```

## Automated Testing

### Unit Tests

```yaml
# .github/workflows/test.yml
name: Platform Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Validate YAML Syntax
        run: |
          pip install yamllint
          yamllint platform/

      - name: Validate Platform Config
        run: pf validate -f platform/platform.yaml --strict

      - name: Test Policies
        run: pf policy test --all --verbose

  integration-tests:
    runs-on: ubuntu-latest
    needs: unit-tests
    steps:
      - uses: actions/checkout@v4

      - name: Setup Kind Cluster
        uses: helm/kind-action@v1

      - name: Deploy Test Platform
        run: |
          pf apply -f platform/platform.yaml \
            --environment=test \
            --auto-approve

      - name: Run Tests
        run: |
          pf test integration --environment=test
          pf health check --platform=test-platform
```

### Policy Tests

```yaml
# tests/policy-tests.yml
name: Policy Compliance

on: [push]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Security Policy Check
        run: |
          pf policy eval \
            --policy=security-baseline \
            --input=platform/ \
            --format=junit \
            --output=security-results.xml

      - name: Publish Results
        uses: mikepenz/action-junit-report@v4
        with:
          report_paths: 'security-results.xml'
```

## Secrets Management in CI/CD

### GitHub Actions Secrets

```yaml
- name: Configure Secrets
  run: |
    pf secrets set db-password --value="${{ secrets.DB_PASSWORD }}"
    pf secrets set api-key --value="${{ secrets.API_KEY }}"
```

### Using Vault

```yaml
- name: Import Secrets from Vault
  uses: hashicorp/vault-action@v2
  with:
    url: https://vault.example.com
    method: jwt
    role: github-actions
    secrets: |
      secret/data/platform db_password | DB_PASSWORD ;
      secret/data/platform api_key | API_KEY

- name: Apply with Secrets
  run: pf apply -f platform/platform.yaml --auto-approve
  env:
    DB_PASSWORD: ${{ env.DB_PASSWORD }}
    API_KEY: ${{ env.API_KEY }}
```

## Notifications

### Slack Notifications

```yaml
- name: Notify Slack on Success
  if: success()
  uses: slackapi/slack-github-action@v1
  with:
    channel-id: 'platform-deploys'
    slack-message: |
      :white_check_mark: Platform deployed successfully
      Branch: ${{ github.ref_name }}
      Commit: ${{ github.sha }}
  env:
    SLACK_BOT_TOKEN: ${{ secrets.SLACK_BOT_TOKEN }}
```

## Best Practices

1. **Always validate first** - Run `pf validate` before plan/apply
2. **Use environments** - Separate staging and production
3. **Require approvals** - Manual approval for production
4. **Test policies** - Run policy tests in CI
5. **Store plans** - Archive plans as artifacts
6. **Notify on failure** - Alert team on deployment failures
7. **Use secrets securely** - Never log secrets

## Next Steps

- [GitOps Guide](../guides/gitops.md)
- [Policies Guide](../guides/policies.md)
- [Secrets Management](../guides/secrets.md)
