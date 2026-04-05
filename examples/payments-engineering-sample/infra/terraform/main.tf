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

resource "kubernetes_namespace" "payments" {
  metadata {
    name = "payments"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-CHECKOUT"
    }
  }
}

resource "kubernetes_namespace" "risk" {
  metadata {
    name = "risk"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-RISK-SCORING"
    }
  }
}

resource "kubernetes_namespace" "flux_system" {
  metadata {
    name = "flux-system"
    annotations = {
      "engmodel.dev/owner-unit" = "FU-GITOPS-OPERATIONS"
    }
  }
}
