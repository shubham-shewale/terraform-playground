# AWS CSPM Monitor - Complete Architecture Diagram

## 🏗️ **AWS CSPM Monitor v3.0.1 Architecture**

```
================================================================================
                        AWS CSPM Monitor v3.0.1 Architecture
================================================================================

┌─────────────────────────────────────────────────────────────────────────────┐
│                              EXTERNAL USERS                                 │
└─────────────────────┬───────────────────────────────────────────────────────┘
                      │
                      ▼ HTTPS (443)
┌─────────────────────────────────────────────────────────────────────────────┐
│                           AMAZON CLOUDFRONT                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Distribution: cspm-monitor.example.com                             │   │
│  │ Origin: S3 Static Website                                          │   │
│  │ Behaviors: /api/* → API Gateway                                     │   │
│  │ Security: Custom SSL Certificate                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │ HTTPS (443)
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AMAZON S3                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Bucket: cspm-monitor-website                                        │   │
│  │ Static Website Hosting: Enabled                                     │   │
│  │ Files: index.html, app.js, style.css                               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼ HTTPS (443) + WAF
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AWS WAF v2                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Web ACL: cspm-monitor-api-waf                                       │   │
│  │ Rules: AWSManagedRulesCommonRuleSet + RateLimit(2000/5min)         │   │
│  │ Associated: API Gateway Stage                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼ HTTPS (443)
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AMAZON API GATEWAY                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ REST API: cspm-monitor-api                                         │   │
│  │ Stage: prod (with caching)                                         │   │
│  │ Endpoints:                                                         │   │
│  │   • GET /findings - Query security findings                        │   │
│  │   • GET /summary - Get findings summary                            │   │
│  │   • GET /health - Health check                                     │   │
│  │ Security Headers: CSP, HSTS, X-Frame-Options, X-Content-Type-Options│   │
│  │ Usage Plan: 50K requests/day, 100 req/sec burst                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │ AWS_PROXY Integration
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AWS LAMBDA (API)                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Function: cspm-monitor-api                                         │   │
│  │ Runtime: Python 3.9                                                 │   │
│  │ Memory: 256MB, Timeout: 30s                                        │   │
│  │ VPC: cspm-monitor-vpc (Private Subnets)                            │   │
│  │ Security Group: Restricted DNS Egress                              │   │
│  │ Environment Variables:                                             │   │
│  │   • DYNAMODB_TABLE_PARAM: /cspm-monitor/dynamodb-table-name        │   │
│  │ Provisioned Concurrency: 2                                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │ HTTPS (443) + DynamoDB API
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AMAZON DYNAMODB                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Table: cspm-monitor-findings                                       │   │
│  │ Primary Key: id (String)                                           │   │
│  │ GSI: SeverityTimestampIndex (severity, timestamp)                 │   │
│  │ Billing: PAY_PER_REQUEST                                           │   │
│  │ Encryption: AES256 (KMS)                                           │   │
│  │ Point-in-Time Recovery: Enabled                                    │   │
│  │ TTL: Enabled (attribute: ttl_timestamp)                           │   │
│  │ Streams: Disabled                                                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      ▲
                      │ HTTPS (443) + Security Hub API
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AWS LAMBDA (SCANNER)                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Function: cspm-monitor-scanner                                     │   │
│  │ Runtime: Python 3.9                                                 │   │
│  │ Memory: 256MB, Timeout: 300s                                       │   │
│  │ VPC: cspm-monitor-vpc (Private Subnets)                            │   │
│  │ Security Group: Restricted DNS Egress                              │   │
│  │ Environment Variables:                                             │   │
│  │   • DYNAMODB_TABLE_PARAM: /cspm-monitor/dynamodb-table-name        │   │
│  │   • SNS_TOPIC_ARN_PARAM: /cspm-monitor/sns-topic-arn               │   │
│  │   • DYNAMODB_TTL_DAYS: 90                                          │   │
│  │ Provisioned Concurrency: 1                                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      ▲
                      │ Event Pattern + JSON
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AMAZON EVENTBRIDGE                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Rule: cspm-monitor-security-hub-findings                           │   │
│  │ Event Pattern:                                                      │   │
│  │   • Source: aws.securityhub                                        │   │
│  │   • Detail-Type: Security Hub Findings - Imported                 │   │
│  │   • Severity: CRITICAL, HIGH, MEDIUM                               │   │
│  │ Targets:                                                           │   │
│  │   • Lambda: cspm-monitor-scanner (with retry)                     │   │
│  │   • DLQ: cspm-monitor-eventbridge-dlq                              │   │
│  │ Schedule Rules:                                                    │   │
│  │   • cspm-monitor-archival-schedule (daily)                         │   │
│  │   • cspm-monitor-sync-schedule (6 hours)                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      ▲
                      │ Security Findings (JSON)
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AWS SECURITY HUB                                 │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Account: Current AWS Account                                       │   │
│  │ Standards: CIS AWS Foundations Benchmark v1.4.0                   │   │
│  │ Integrations:                                                      │   │
│  │   • GuardDuty (threat detection)                                   │   │
│  │   • Inspector (vulnerability assessment)                           │   │
│  │   • Macie (data security)                                          │   │
│  │   • Config (resource compliance)                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      ▲
                      │ Security Events
┌─────────────────────────────────────────────────────────────────────────────┐
│                    AWS INTEGRATED SERVICES                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │ GuardDuty   │  │ Inspector   │  │   Macie    │  │   Config    │     │
│  │ (Threats)   │  │ (Vulns)     │  │ (Data Sec) │  │ (Compliance)│     │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
```

```
================================================================================
                        DATA LIFECYCLE & ARCHIVAL
================================================================================

┌─────────────────────────────────────────────────────────────────────────────┐
│                         AWS LAMBDA (ARCHIVER)                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Function: cspm-monitor-archiver                                    │   │
│  │ Runtime: Python 3.9                                                 │   │
│  │ Memory: 512MB, Timeout: 900s                                       │   │
│  │ VPC: cspm-monitor-vpc (Private Subnets)                            │   │
│  │ Security Group: Restricted DNS Egress                              │   │
│  │ Environment Variables:                                             │   │
│  │   • DYNAMODB_TABLE_PARAM: /cspm-monitor/dynamodb-table-name        │   │
│  │   • S3_ARCHIVE_BUCKET: cspm-monitor-security-archive-{account}     │   │
│  │   • RETENTION_DAYS: 90                                             │   │
│  │ Trigger: EventBridge Schedule (daily)                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │ HTTPS (443) + S3 API
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AMAZON S3 (ARCHIVE)                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Bucket: cspm-monitor-security-archive-{account}                    │   │
│  │ Versioning: Enabled                                                │   │
│  │ Encryption: AES256                                                 │   │
│  │ Public Access: Blocked                                             │   │
│  │ MFA Delete: Required for delete operations                        │   │
│  │ Lifecycle Rules:                                                   │   │
│  │   • 30 days: Standard → Standard-IA                               │   │
│  │   • 90 days: Standard-IA → Glacier                                │   │
│  │   • 365 days: Glacier → Deep Archive                              │   │
│  │   • 2555 days: Delete                                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      ▲
                      │ HTTPS (443) + DynamoDB API
┌─────────────────────────────────────────────────────────────────────────────┐
│                         AMAZON DYNAMODB                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Table: cspm-monitor-findings                                       │   │
│  │ TTL: Automatic deletion after 90 days                              │   │
│  │ Backup: AWS Backup (daily, 35-day retention)                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

```
================================================================================
                        MONITORING & ALERTING
================================================================================

┌─────────────────────────────────────────────────────────────────────────────┐
│                        AMAZON CLOUDWATCH                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Log Groups:                                                        │   │
│  │   • /aws/lambda/cspm-monitor-scanner (30 days)                     │   │
│  │   • /aws/lambda/cspm-monitor-api (30 days)                         │   │
│  │   • /aws/lambda/cspm-monitor-archiver (30 days)                    │   │
│  │   • /aws/apigateway/cspm-monitor-api (30 days)                     │   │
│  │ Dashboards:                                                        │   │
│  │   • cspm-monitor-dashboard (with conditional archiver metrics)    │   │
│  │ Alarms:                                                            │   │
│  │   • Lambda Errors (>5 in 15min)                                   │   │
│  │   • API Gateway 4XX/5XX (>100 in 5min)                            │   │
│  │   • DynamoDB Throttling (>10 in 5min)                             │   │
│  │   • Critical Findings (>5 in 5min)                                │   │
│  │ Query Definitions:                                                 │   │
│  │   • Error Analysis, Security Findings Analysis                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼ Alarm Actions
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AMAZON SNS                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Topics:                                                            │   │
│  │   • cspm-monitor-alerts (standard notifications)                   │   │
│  │   • cspm-monitor-critical-alerts (escalation)                      │   │
│  │ Subscriptions: Email, SMS, PagerDuty integration                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

```
================================================================================
                        CONFIGURATION & SECURITY
================================================================================

┌─────────────────────────────────────────────────────────────────────────────┐
│                   AWS SYSTEMS MANAGER PARAMETER STORE                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Parameters:                                                        │   │
│  │   • /cspm-monitor/dynamodb-table-name (SecureString)               │   │
│  │   • /cspm-monitor/sns-topic-arn (SecureString)                      │   │
│  │ Usage: Runtime configuration for Lambda functions                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AWS KMS (KEY MANAGEMENT)                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Keys:                                                              │   │
│  │   • DynamoDB Encryption Key                                        │   │
│  │   • S3 Archive Encryption Key                                      │   │
│  │   • CloudFront SSL Certificate                                     │   │
│  │ Usage: Server-side encryption for data at rest                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AWS BACKUP                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Vault: cspm-monitor-security-logs-backup                           │   │
│  │ Plan: Daily backup at 5 AM UTC                                     │   │
│  │ Retention: 35 days                                                 │   │
│  │ Resources: DynamoDB table                                          │   │
│  │ Cross-Region: Optional                                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

```
================================================================================
                        NETWORKING & SECURITY
================================================================================

┌─────────────────────────────────────────────────────────────────────────────┐
│                          AMAZON VPC                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ VPC: cspm-monitor-vpc (10.0.0.0/16)                                │   │
│  │ Subnets:                                                           │   │
│  │   • 10.0.0.0/24 (us-east-1a)                                       │   │
│  │   • 10.0.1.0/24 (us-east-1b)                                       │   │
│  │ Internet Gateway: Attached                                         │   │
│  │ Route Tables: Public routing                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      SECURITY GROUPS                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Lambda Security Group:                                             │   │
│  │   • Egress:                                                        │   │
│  │     - HTTPS (443) to 0.0.0.0/0 (AWS API calls)                     │   │
│  │     - DNS TCP (53) to 169.254.169.253/32 (AWS DNS)                 │   │
│  │     - DNS UDP (53) to 169.254.169.253/32 (AWS DNS)                 │   │
│  │   • Ingress: None (VPC-only)                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 📋 **Service Communication Matrix**

| Source Service | Target Service | Protocol | Port | Purpose |
|----------------|----------------|----------|------|---------|
| **Users** | CloudFront | HTTPS | 443 | Web access |
| **CloudFront** | S3 | HTTPS | 443 | Static content |
| **CloudFront** | API Gateway | HTTPS | 443 | API requests |
| **API Gateway** | Lambda | HTTP | - | Function invocation |
| **Lambda** | DynamoDB | HTTPS | 443 | Data operations |
| **Lambda** | Security Hub | HTTPS | 443 | Findings retrieval |
| **Lambda** | S3 | HTTPS | 443 | Archive operations |
| **Lambda** | SNS | HTTPS | 443 | Alert notifications |
| **Lambda** | Systems Manager | HTTPS | 443 | Configuration |
| **Security Hub** | EventBridge | HTTPS | 443 | Event publishing |
| **EventBridge** | Lambda | HTTPS | 443 | Event delivery |
| **EventBridge** | SQS | HTTPS | 443 | Dead letter queue |
| **All Services** | CloudWatch | HTTPS | 443 | Logging & metrics |
| **CloudWatch** | SNS | HTTPS | 443 | Alarm notifications |

## 🔄 **Data Flow Summary**

### **Primary Data Flow**
1. **Security Events** → AWS Services → Security Hub → EventBridge → Lambda Scanner
2. **Storage** → Lambda Scanner → DynamoDB (with TTL)
3. **User Access** → CloudFront → S3/API Gateway → Lambda API → DynamoDB
4. **Archival** → EventBridge Schedule → Lambda Archiver → S3 Archive
5. **Cleanup** → DynamoDB TTL → Automatic deletion

### **Monitoring Flow**
1. **Logs** → All Services → CloudWatch Logs
2. **Metrics** → Services → CloudWatch Metrics
3. **Alarms** → CloudWatch → SNS → Notifications
4. **Dashboards** → CloudWatch → Visual monitoring

### **Configuration Flow**
1. **Parameters** → Systems Manager → Lambda functions
2. **Encryption** → KMS → Data at rest
3. **Backups** → AWS Backup → Automated snapshots

## 🏷️ **Key Architecture Features**

### **Security Features**
- ✅ **WAF Protection**: Rate limiting and managed rules
- ✅ **VPC Deployment**: Lambda functions in private subnets
- ✅ **Encryption**: AES256 for all data at rest
- ✅ **Access Control**: Least privilege IAM policies
- ✅ **Network Security**: Restricted DNS egress rules

### **Performance Features**
- ✅ **Provisioned Concurrency**: Sub-second cold starts
- ✅ **API Caching**: Response caching in API Gateway
- ✅ **Optimized Queries**: GSI for efficient data retrieval
- ✅ **Connection Pooling**: Efficient AWS SDK usage

### **Reliability Features**
- ✅ **Error Handling**: Comprehensive exception handling
- ✅ **Retry Policies**: EventBridge retry with exponential backoff
- ✅ **Dead Letter Queues**: Failed message handling
- ✅ **Transactional Operations**: Safe archival with rollback

### **Compliance Features**
- ✅ **Audit Logging**: CloudTrail integration
- ✅ **Data Retention**: Configurable retention policies
- ✅ **Backup Strategy**: Automated daily backups
- ✅ **Access Logging**: Comprehensive API logging

## 📊 **Service Dependencies**

```
CloudFront
├── S3 (Static Website)
└── API Gateway (API requests)

API Gateway
├── WAF v2 (Protection)
├── Lambda API (Backend)
└── CloudWatch (Access Logs)

Lambda API
├── DynamoDB (Data storage)
├── Systems Manager (Configuration)
└── CloudWatch (Logs/Metrics)

Lambda Scanner
├── Security Hub (Findings)
├── DynamoDB (Storage)
├── SNS (Alerts)
├── Systems Manager (Configuration)
└── CloudWatch (Logs/Metrics)

Lambda Archiver
├── DynamoDB (Source data)
├── S3 (Archive storage)
├── Systems Manager (Configuration)
└── CloudWatch (Logs/Metrics)

EventBridge
├── Security Hub (Event source)
├── Lambda Scanner (Target)
├── Lambda Archiver (Scheduled target)
└── SQS (DLQ)

DynamoDB
├── KMS (Encryption)
├── AWS Backup (Snapshots)
└── CloudWatch (Metrics)

S3 Archive
├── KMS (Encryption)
└── CloudWatch (Access Logs)
```

This comprehensive diagram shows the complete AWS architecture of the CSPM Monitor v3.0.1, including all services, their configurations, communication patterns, and dependencies.