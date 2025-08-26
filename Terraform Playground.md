# Terraform Playground

## Overview

The **Terraform Playground** repository is a collection of Terraform configurations designed for learning, experimentation, and demonstration of various infrastructure-as-code (IaC) concepts. It provides modular examples for building foundational cloud infrastructure, deploying secure bastion hosts, and hosting static websites on AWS.

This repository is structured into multiple projects, each focusing on a specific use case or architecture. It is ideal for developers, DevOps engineers, and cloud practitioners looking to explore Terraform's capabilities in a hands-on manner.

---

## Repository Structure

### **1. Basic VPC**
- **Purpose**: Demonstrates how to create a basic Virtual Private Cloud (VPC) with public and private subnets, EC2 instances, and Systems Manager (SSM) integration.
- **Key Features**:
  - Public and private subnets across multiple availability zones.
  - EC2 instances with SSM Agent for remote management.
  - Backend configuration for remote state storage.
- **Diagram**: `basic-vpc.png`

### **2. Bastion Host**
- **Purpose**: Provides a secure bastion host setup for accessing private instances within a VPC.
- **Key Features**:
  - Modular design with reusable components:
    - **Bastion**: Provisions the bastion host.
    - **Key Pair**: Generates SSH key pairs for secure access.
    - **Private Instance**: Deploys private EC2 instances.
    - **Security Group**: Configures security groups for controlled access.
    - **VPC**: Creates the underlying VPC infrastructure.
  - Outputs for SSH connection details and private instance information.

### **3. Static Website**
- **Purpose**: Deploys a static website hosted on AWS S3 with optional CloudFront distribution.
- **Key Features**:
  - S3 bucket for hosting static files.
  - Optional CloudFront distribution for content delivery.
  - Outputs for the website URL and CloudFront domain (if enabled).
- **Diagram**: `static-website.png`

---

## Key Features of the Repository
- **Modular Design**: Reusable modules for common infrastructure components.
- **Remote State Management**: Backend configuration for storing Terraform state files securely.
- **Diagrams**: Visual representations of the architecture for better understanding.
- **GitHub Actions**: CI/CD workflows for automating Terraform operations.

---

## Prerequisites
- Terraform CLI installed on your local machine.
- AWS CLI configured with appropriate credentials.
- Basic knowledge of Terraform and AWS.

---

## Usage
1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/terraform-playground.git
   cd terraform-playground