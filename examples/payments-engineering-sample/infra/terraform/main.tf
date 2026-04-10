terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.30"
    }
  }
}

provider "aws" {
  region = "eu-north-1"
}

locals {
  environment  = "prod"
  cluster_name = "payments-prod-eks"
}

# Minimal illustrative EKS control plane declaration.
# engmodel:runtime-description: provisions the shared EKS control plane that hosts payments and risk workloads
resource "aws_eks_cluster" "payments" {
  name     = "payments-prod-eks"
  role_arn = "arn:aws:iam::123456789012:role/eks-cluster-role"
  version  = "1.30"
  tags = {
    "engmodel.dev/owner-unit" = "FU-CLUSTER-PROVISIONING"
  }

  vpc_config {
    subnet_ids = ["subnet-aaaa1111", "subnet-bbbb2222"]
  }
}

# engmodel:runtime-description: runtime namespace boundary for checkout and payment authorization workloads
resource "kubernetes_namespace" "payments" {
  metadata {
    name = "payments"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-CHECKOUT"
    }
  }
}

# engmodel:runtime-description: runtime namespace boundary for fraud scoring and review support workloads
resource "kubernetes_namespace" "risk" {
  metadata {
    name = "risk"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-RISK-SCORING"
    }
  }
}

# engmodel:runtime-description: control-plane namespace where Flux GitOps operators reconcile deployment state
resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = "flux-system"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-GITOPS-OPERATIONS"
    }
  }
}
