# Complete Cloud & VPS Provider Reference

**Last Updated**: January 23, 2026
**nself Version**: 0.4.7+

---

## Overview

nself supports deployment to any server accessible via SSH. This document lists all supported and planned providers with their characteristics, pricing, and optimal use cases.

> **Note**: Pricing information reflects January 2026 rates and may change. Always verify current pricing on provider websites.

---

## Provider Categories

| Category | Count | Description |
|----------|-------|-------------|
| **Major Cloud** | 5 | Enterprise-grade, global infrastructure |
| **Developer Cloud** | 5 | Developer-focused, simple pricing |
| **Budget EU** | 5 | European providers, excellent value |
| **Budget Global** | 4 | Global budget options |
| **Regional/Specialty** | 4 | Region-specific or compliance-focused |
| **Extreme Budget** | 3 | Lowest cost options |
| **Total** | **26** | All supported providers |

---

## Major Cloud Providers (5)

### AWS (Amazon Web Services)

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 30+ worldwide |
| **Smallest Instance** | t3.micro (1 vCPU, 1GB) |
| **Starting Price** | ~$8/mo (t3.micro) |
| **Free Tier** | 750 hours/mo for 12 months |
| **Best For** | Enterprise, complex architectures |
| **Egress Pricing** | $0.09/GB after 100GB |

**nself Commands**:
```bash
nself providers init aws
nself provision aws --size small
nself provision aws --size medium --region us-east-1
```

**Strengths**: Most services, best documentation, largest ecosystem
**Weaknesses**: Complex pricing, expensive egress, steep learning curve

---

### GCP (Google Cloud Platform)

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 35+ worldwide |
| **Smallest Instance** | e2-micro (0.25 vCPU, 1GB) |
| **Starting Price** | ~$6/mo (e2-micro) |
| **Free Tier** | e2-micro always free |
| **Best For** | ML/AI workloads, data analytics |
| **Egress Pricing** | $0.12/GB |

**nself Commands**:
```bash
nself providers init gcp
nself provision gcp --size small
nself provision gcp --size medium --region us-central1
```

**Strengths**: Best ML/AI tools, generous free tier, great networking
**Weaknesses**: Complex IAM, fewer managed services than AWS

---

### Azure (Microsoft Azure)

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 60+ worldwide |
| **Smallest Instance** | B1s (1 vCPU, 1GB) |
| **Starting Price** | ~$7/mo (B1s) |
| **Free Tier** | 750 hours B1s for 12 months |
| **Best For** | Microsoft stack, enterprise |
| **Egress Pricing** | $0.087/GB |

**nself Commands**:
```bash
nself providers init azure
nself provision azure --size small
nself provision azure --size medium --region eastus
```

**Strengths**: Best Windows support, enterprise integrations, hybrid cloud
**Weaknesses**: Portal UX, complex naming, support response times

---

### Oracle Cloud Infrastructure (OCI)

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 40+ worldwide |
| **Smallest Instance** | VM.Standard.E2.1.Micro |
| **Starting Price** | **FREE** (Always Free tier) |
| **Free Tier** | 2 AMD VMs + 4 ARM cores FOREVER |
| **Best For** | Budget workloads, ARM development |
| **Egress Pricing** | 10TB/mo FREE |

**nself Commands** (v0.4.7):
```bash
nself providers init oracle
nself provision oracle --size free    # Always Free tier
nself provision oracle --size small --arm  # ARM instances
```

**Strengths**: Best free tier in industry, excellent ARM instances
**Weaknesses**: Smaller ecosystem, occasional availability issues

---

### IBM Cloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 19 worldwide |
| **Smallest Instance** | bx2-2x8 (2 vCPU, 8GB) |
| **Starting Price** | ~$40/mo |
| **Free Tier** | Limited (Lite accounts) |
| **Best For** | Regulated industries, bare metal |
| **Egress Pricing** | $0.09/GB |

**nself Commands** (v0.4.7):
```bash
nself providers init ibm
nself provision ibm --size medium
nself provision ibm --bare-metal  # Dedicated hardware
```

**Strengths**: Strong compliance, bare metal options, Watson AI
**Weaknesses**: Higher base pricing, smaller community

---

## Developer Cloud Providers (5)

### DigitalOcean

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 15 worldwide |
| **Smallest Instance** | Basic Droplet (1 vCPU, 1GB) |
| **Starting Price** | $6/mo |
| **Free Tier** | $200 credit for 60 days |
| **Best For** | Developers, simple deployments |
| **Egress Pricing** | 1TB included, then $0.01/GB |

**nself Commands**:
```bash
nself providers init digitalocean
nself provision digitalocean --size small
nself provision digitalocean --size medium --region nyc1
```

**Strengths**: Best developer UX, predictable pricing, great docs
**Weaknesses**: Fewer enterprise features, limited regions

---

### Linode (Akamai)

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 11 worldwide |
| **Smallest Instance** | Nanode (1 vCPU, 1GB) |
| **Starting Price** | $5/mo |
| **Free Tier** | $100 credit for 60 days |
| **Best For** | Reliable VPS, good support |
| **Egress Pricing** | 1TB included, then $0.01/GB |

**nself Commands**:
```bash
nself providers init linode
nself provision linode --size small
nself provision linode --size medium --region us-east
```

**Strengths**: Reliable, good support, Akamai CDN integration
**Weaknesses**: Fewer managed services than DO

---

### Vultr

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 32 worldwide |
| **Smallest Instance** | Cloud Compute (1 vCPU, 1GB) |
| **Starting Price** | $5/mo ($2.50 for IPv6-only) |
| **Free Tier** | $250 credit for 30 days |
| **Best For** | Global coverage, bare metal |
| **Egress Pricing** | 1-2TB included |

**nself Commands**:
```bash
nself providers init vultr
nself provision vultr --size small
nself provision vultr --size medium --region ewr  # Newark
nself provision vultr --bare-metal
```

**Strengths**: Most locations (32+), bare metal options, high-frequency compute
**Weaknesses**: UI less polished than DO

---

### Scaleway

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 4 (Paris, Amsterdam, Warsaw, soon more) |
| **Smallest Instance** | DEV1-S (2 vCPU, 2GB) |
| **Starting Price** | €2/mo (STARDUST) |
| **Free Tier** | Limited free resources |
| **Best For** | EU-focused, GDPR compliance |
| **Egress Pricing** | 75GB included |

**nself Commands**:
```bash
nself providers init scaleway
nself provision scaleway --size small
nself provision scaleway --size medium --region fr-par
```

**Strengths**: EU data sovereignty, competitive ARM pricing
**Weaknesses**: Limited regions, smaller ecosystem

---

### UpCloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 13 worldwide |
| **Smallest Instance** | 1 vCPU, 1GB |
| **Starting Price** | $5/mo |
| **Free Tier** | $25 credit |
| **Best For** | High I/O workloads |
| **Egress Pricing** | 1TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init upcloud
nself provision upcloud --size small
nself provision upcloud --maxiops  # High-performance storage
```

**Strengths**: MaxIOPS storage (fastest in class), excellent performance
**Weaknesses**: Fewer locations than Vultr

---

## Budget EU Providers (5)

### Hetzner

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 5 (Germany, Finland, US) |
| **Smallest Instance** | CX11 (1 vCPU, 2GB) |
| **Starting Price** | €3.29/mo (~$3.50) |
| **Free Tier** | None |
| **Best For** | Best price/performance ratio |
| **Egress Pricing** | 20TB included |

**nself Commands**:
```bash
nself providers init hetzner
nself provision hetzner --size small
nself provision hetzner --size medium --region fsn1  # Falkenstein
nself provision hetzner --dedicated  # Dedicated servers
```

**Strengths**: **BEST value in industry**, generous bandwidth, German quality
**Weaknesses**: Support hours limited, no managed K8s

---

### OVHcloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 30+ worldwide |
| **Smallest Instance** | Starter (1 vCPU, 2GB) |
| **Starting Price** | €3.50/mo (~$4) |
| **Free Tier** | None |
| **Best For** | High bandwidth, bare metal |
| **Egress Pricing** | Unlimited in most plans |

**nself Commands**:
```bash
nself providers init ovh
nself provision ovh --size small
nself provision ovh --size medium --region gra  # Gravelines
nself provision ovh --dedicated
```

**Strengths**: Unlimited bandwidth, huge bare metal selection, DDoS protection
**Weaknesses**: Complex interface, slower support

---

### IONOS

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.5) |
| **Regions** | 10+ (EU, US) |
| **Smallest Instance** | VPS S (1 vCPU, 512MB) |
| **Starting Price** | $2/mo |
| **Free Tier** | None |
| **Best For** | Entry-level VPS, European hosting |
| **Egress Pricing** | Unlimited |

**nself Commands**:
```bash
nself providers init ionos
nself provision ionos --size small
nself provision ionos --size medium --region de
```

**Strengths**: Very cheap entry, unlimited traffic, Plesk included
**Weaknesses**: Limited advanced features

---

### Contabo

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 9 (EU, US, Asia, Australia) |
| **Smallest Instance** | Cloud VPS S (4 vCPU, 8GB) |
| **Starting Price** | €4.50/mo (~$5) |
| **Free Tier** | None |
| **Best For** | Maximum specs per dollar |
| **Egress Pricing** | 32TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init contabo
nself provision contabo --size small   # Actually 4 vCPU, 8GB!
nself provision contabo --size medium --region eu
```

**Strengths**: **Most specs for lowest price** - 4 vCPU/8GB for ~$5
**Weaknesses**: Slower support, performance can vary, setup fees

---

### Netcup

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 3 (Germany, Austria, USA) |
| **Smallest Instance** | VPS 200 G10s (2 vCPU, 2GB) |
| **Starting Price** | €2.99/mo (~$3.25) |
| **Free Tier** | None |
| **Best For** | German quality at budget price |
| **Egress Pricing** | ~80TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init netcup
nself provision netcup --size small
nself provision netcup --size medium --region de
```

**Strengths**: Excellent German quality, great value, reliable
**Weaknesses**: German-focused, limited English support

---

## Budget Global Providers (4)

### Hostinger VPS

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 8 (US, EU, Asia, South America) |
| **Smallest Instance** | KVM 1 (1 vCPU, 4GB) |
| **Starting Price** | $4.99/mo |
| **Free Tier** | None |
| **Best For** | Beginners, good UI |
| **Egress Pricing** | 1TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init hostinger
nself provision hostinger --size small
nself provision hostinger --size medium --region us
```

**Strengths**: Great UI, beginner-friendly, decent specs
**Weaknesses**: Renewal prices higher, limited advanced features

---

### Hostwinds

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 3 (Seattle, Dallas, Amsterdam) |
| **Smallest Instance** | Unmanaged VPS (1 vCPU, 1GB) |
| **Starting Price** | $4.99/mo |
| **Free Tier** | None |
| **Best For** | Managed VPS on budget |
| **Egress Pricing** | 1TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init hostwinds
nself provision hostwinds --size small
nself provision hostwinds --managed  # Managed option
```

**Strengths**: 24/7 support, managed VPS option, nightly backups
**Weaknesses**: Fewer locations

---

### Kamatera

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 22 worldwide |
| **Smallest Instance** | 1 vCPU, 1GB |
| **Starting Price** | $4/mo |
| **Free Tier** | 30-day free trial |
| **Best For** | Custom configurations |
| **Egress Pricing** | 1TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init kamatera
nself provision kamatera --size small
nself provision kamatera --custom --cpu 4 --ram 8 --disk 100
```

**Strengths**: Highly configurable, many locations, 30-day trial
**Weaknesses**: Per-hour billing adds up, no flat monthly

---

### SSD Nodes

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 4 (US, EU) |
| **Smallest Instance** | 2 vCPU, 4GB |
| **Starting Price** | $4.99/mo |
| **Free Tier** | None |
| **Best For** | Nested virtualization, price locks |
| **Egress Pricing** | 4TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init ssdnodes
nself provision ssdnodes --size small
nself provision ssdnodes --nested-virt  # For testing K8s locally
```

**Strengths**: Price locks (no renewal increase), nested virtualization
**Weaknesses**: Fewer locations, occasional overselling

---

## Regional/Specialty Providers (4)

### Exoscale

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 4 (Switzerland, Germany, Austria, Bulgaria) |
| **Smallest Instance** | Tiny (1 vCPU, 512MB) |
| **Starting Price** | ~$5/mo |
| **Free Tier** | None |
| **Best For** | Swiss privacy, GDPR compliance |
| **Egress Pricing** | Per GB |

**nself Commands** (v0.4.7):
```bash
nself providers init exoscale
nself provision exoscale --size small
nself provision exoscale --size medium --region ch-gva-2  # Geneva
```

**Strengths**: Swiss data protection, excellent compliance, S3-compatible storage
**Weaknesses**: Higher pricing than budget EU options

---

### Alibaba Cloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 28 worldwide (strongest in Asia) |
| **Smallest Instance** | ecs.t6-c1m1.large |
| **Starting Price** | ~$5/mo |
| **Free Tier** | Trial credits available |
| **Best For** | Asia-Pacific, China market |
| **Egress Pricing** | Varies by region |

**nself Commands** (v0.4.7):
```bash
nself providers init alibaba
nself provision alibaba --size small
nself provision alibaba --size medium --region cn-hangzhou
nself provision alibaba --size medium --region ap-southeast-1  # Singapore
```

**Strengths**: Best Asia coverage, access to China regions
**Weaknesses**: Complex pricing, documentation quality varies

---

### Tencent Cloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 25+ (strong in Asia) |
| **Smallest Instance** | S5.SMALL2 (1 vCPU, 2GB) |
| **Starting Price** | ~$4/mo |
| **Free Tier** | Trial available |
| **Best For** | China market, gaming |
| **Egress Pricing** | Varies |

**nself Commands** (v0.4.7):
```bash
nself providers init tencent
nself provision tencent --size small
nself provision tencent --size medium --region ap-guangzhou
```

**Strengths**: Strong in China, good gaming infrastructure
**Weaknesses**: Less documentation in English

---

### Yandex Cloud

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 3 (Russia) |
| **Smallest Instance** | standard-v3 (2 vCPU, 2GB) |
| **Starting Price** | ~$5/mo |
| **Free Tier** | Trial credits |
| **Best For** | Russian market |
| **Egress Pricing** | Per GB |

**nself Commands** (v0.4.7):
```bash
nself providers init yandex
nself provision yandex --size small
nself provision yandex --size medium --region ru-central1
```

**Strengths**: Best for Russian deployments
**Weaknesses**: Limited to Russia, sanctions considerations

---

## Extreme Budget Providers (3)

### RackNerd

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 7 (US primarily) |
| **Smallest Instance** | 1 vCPU, 768MB |
| **Starting Price** | $1.98/mo (promotional) |
| **Free Tier** | None |
| **Best For** | Absolute minimum cost |
| **Egress Pricing** | 1-2TB included |

**nself Commands** (v0.4.7):
```bash
nself providers init racknerd
nself provision racknerd --size small
```

**Strengths**: Cheapest in market, good for testing
**Weaknesses**: You get what you pay for, overselling possible

---

### BuyVM

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 3 (US, Luxembourg) |
| **Smallest Instance** | Slice 512 (1 vCPU, 512MB) |
| **Starting Price** | $2/mo |
| **Free Tier** | None |
| **Best For** | Small projects, testing |
| **Egress Pricing** | Unmetered |

**nself Commands** (v0.4.7):
```bash
nself providers init buyvm
nself provision buyvm --size small
```

**Strengths**: Unmetered bandwidth, DDoS protection, reliable for price
**Weaknesses**: Limited availability, often sold out

---

### Time4VPS

| Attribute | Value |
|-----------|-------|
| **Status** | ✅ Supported (v0.4.7) |
| **Regions** | 1 (Lithuania) |
| **Smallest Instance** | Linux 1 (1 vCPU, 1GB) |
| **Starting Price** | €2.99/mo (~$3.25) |
| **Free Tier** | None |
| **Best For** | Budget EU hosting |
| **Egress Pricing** | Unmetered |

**nself Commands** (v0.4.7):
```bash
nself providers init time4vps
nself provision time4vps --size small
```

**Strengths**: Very affordable, EU location, unmetered bandwidth
**Weaknesses**: Single location, basic support

---

## Provider Comparison Matrix

### Price/Performance Leaders

| Rank | Provider | Best For | Starting Price | Our Rating |
|------|----------|----------|----------------|------------|
| 1 | **Hetzner** | Best overall value | €3.29/mo | ⭐⭐⭐⭐⭐ |
| 2 | **Contabo** | Maximum specs | €4.50/mo | ⭐⭐⭐⭐ |
| 3 | **Oracle Cloud** | Free tier | FREE | ⭐⭐⭐⭐⭐ |
| 4 | **Netcup** | German quality budget | €2.99/mo | ⭐⭐⭐⭐ |
| 5 | **Vultr** | Global coverage | $5/mo | ⭐⭐⭐⭐ |
| 6 | **DigitalOcean** | Developer experience | $6/mo | ⭐⭐⭐⭐⭐ |
| 7 | **Linode** | Reliability | $5/mo | ⭐⭐⭐⭐ |

### By Use Case

| Use Case | Recommended Provider | Why |
|----------|---------------------|-----|
| **Learning/Testing** | Oracle Cloud (Free), RackNerd | Lowest cost |
| **Side Projects** | Hetzner, Contabo | Best value |
| **Startups** | DigitalOcean, Vultr | Great UX + docs |
| **Production (US)** | AWS, DigitalOcean, Vultr | Reliability + support |
| **Production (EU)** | Hetzner, OVH, Scaleway | GDPR + value |
| **Production (Asia)** | Alibaba, Vultr | Regional coverage |
| **Enterprise** | AWS, GCP, Azure | Compliance + scale |
| **High Bandwidth** | OVH, Hetzner, BuyVM | Unmetered/cheap egress |
| **Bare Metal** | Hetzner, OVH, Vultr | Dedicated hardware |

### Managed Kubernetes Availability

| Provider | Managed K8s | Name |
|----------|-------------|------|
| AWS | ✅ | EKS |
| GCP | ✅ | GKE |
| Azure | ✅ | AKS |
| DigitalOcean | ✅ | DOKS |
| Linode | ✅ | LKE |
| Vultr | ✅ | VKE |
| Scaleway | ✅ | Kapsule |
| OVH | ✅ | OVHcloud Kubernetes |
| Oracle | ✅ | OKE |
| Hetzner | ❌ | Use k3s |
| Contabo | ❌ | Use k3s |
| IONOS | ✅ | IONOS K8s |

---

## Provider Configuration Files

All provider configurations are stored in:
```
~/.nself/providers/
├── aws.yml
├── gcp.yml
├── azure.yml
├── digitalocean.yml
├── hetzner.yml
├── linode.yml
├── vultr.yml
├── ionos.yml
├── ovh.yml
├── scaleway.yml
└── ... (additional providers)
```

### Configuration Example

```yaml
# ~/.nself/providers/hetzner.yml
provider: hetzner
api_token: "your-api-token"
default_region: fsn1
default_size: small
ssh_key_name: nself-deploy

size_mappings:
  small:
    type: cx11
    vcpu: 1
    ram: 2GB
    disk: 20GB
  medium:
    type: cx21
    vcpu: 2
    ram: 4GB
    disk: 40GB
  large:
    type: cx31
    vcpu: 2
    ram: 8GB
    disk: 80GB
```

---

## Adding New Providers

nself can deploy to any server with SSH access. For providers not yet integrated:

```bash
# Manual server setup
nself servers add myserver --ip 1.2.3.4 --user root --key ~/.ssh/id_rsa

# Deploy to manual server
nself deploy myserver
```

For full provider integration (API-based provisioning), see the [Provider Development Guide](./PROVIDER-DEVELOPMENT.md).

---

## Related Documentation

- [Provider Commands](../commands/PROVIDERS.md) - CLI reference
- [Provision Command](../commands/PROVISION.md) - Infrastructure provisioning
- [Deploy Command](../commands/DEPLOY.md) - Deployment operations
- [Sync Command](../commands/SYNC.md) - Environment synchronization

---

*Last Updated: January 23, 2026*
