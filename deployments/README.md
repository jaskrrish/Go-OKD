# QKD System Deployment Guide

This guide explains how to deploy the production-grade Quantum Key Distribution (QKD) system using Docker.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Services](#services)
- [Monitoring](#monitoring)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Docker 20.10+ and Docker Compose 2.0+
- Minimum 4GB RAM, 20GB disk space
- (Optional) IBM Quantum Platform API key for real quantum hardware

## Quick Start

### 1. Copy Environment File

```bash
cp .env.qkd.example .env.qkd
```

### 2. Edit Environment Variables

```bash
nano .env.qkd
```

**Important**: Change these security-critical variables:
- `DB_PASSWORD` - PostgreSQL password
- `REDIS_PASSWORD` - Redis password
- `JWT_SECRET` - JWT signing key (min 32 characters)
- `GRAFANA_PASSWORD` - Grafana admin password

### 3. Start Services

```bash
docker-compose -f docker-compose.qkd.yml --env-file .env.qkd up -d
```

### 4. Verify Services

```bash
docker-compose -f docker-compose.qkd.yml ps
```

All services should show status "healthy" or "running".

### 5. Access Services

- **QKD API**: http://localhost:8080
- **Grafana**: http://localhost:3000 (admin/password from .env.qkd)
- **Prometheus**: http://localhost:9091
- **API Documentation**: http://localhost:8080/api/docs

## Configuration

### Environment Variables

See `.env.qkd.example` for all available configuration options.

Key categories:
- **Database**: PostgreSQL connection settings
- **Cache**: Redis configuration
- **Quantum**: IBM Qiskit API settings
- **Security**: JWT, TLS, authentication
- **Performance**: Rate limiting, timeouts
- **Monitoring**: Metrics and logging

### IBM Qiskit Setup (Optional)

To use real quantum hardware:

1. Get API key from https://quantum-computing.ibm.com/
2. Set in `.env.qkd`:
   ```
   QISKIT_API_KEY=your_api_key_here
   QISKIT_BACKEND=ibmq_qasm_simulator
   ```

Available backends:
- `ibmq_qasm_simulator` - Free simulator (default)
- `ibm_kyoto` - Real quantum hardware (requires access)
- `ibm_osaka` - Real quantum hardware (requires access)

## Services

### QKD API (Port 8080)

Main application server providing:
- RESTful API for quantum key distribution
- BB84 protocol implementation
- Session management
- Key storage and retrieval

**Endpoints:**
```
POST   /api/v1/qkd/session/initiate       - Create new session
POST   /api/v1/qkd/session/join           - Join existing session
POST   /api/v1/qkd/session/{id}/execute   - Execute key exchange
GET    /api/v1/qkd/key/{id}                - Retrieve quantum key
DELETE /api/v1/qkd/key/{id}                - Revoke key
GET    /health                             - Health check
GET    /metrics                            - Prometheus metrics
```

### PostgreSQL (Port 5432)

Database for persistent storage:
- Session metadata
- Encrypted key material
- Audit logs
- Performance metrics

**Access:**
```bash
docker exec -it qkd-postgres psql -U qkd_user -d qkd_production
```

### Redis (Port 6379)

Cache layer for:
- Session state
- Temporary data
- Rate limiting
- API response caching

**Access:**
```bash
docker exec -it qkd-redis redis-cli -a $REDIS_PASSWORD
```

### Prometheus (Port 9091)

Metrics collection and monitoring:
- API performance metrics
- QBER statistics
- Session statistics
- System resource usage

**Metrics:**
- `qkd_sessions_active` - Active sessions
- `qkd_sessions_created_total` - Total sessions created
- `qkd_keys_generated_total` - Total keys generated
- `qkd_session_qber` - Quantum bit error rate
- `qkd_http_request_duration_seconds` - HTTP request latency
- `qkd_http_requests_total` - HTTP request count

### Grafana (Port 3000)

Visualization dashboards:
- QKD System Overview
- Performance metrics
- Security alerts
- Resource utilization

**Default dashboard:** QKD System Overview

### Node Exporter (Port 9100)

System-level metrics:
- CPU usage
- Memory usage
- Disk I/O
- Network statistics

## Monitoring

### View Logs

```bash
# All services
docker-compose -f docker-compose.qkd.yml logs -f

# Specific service
docker-compose -f docker-compose.qkd.yml logs -f qkd-api

# Last 100 lines
docker-compose -f docker-compose.qkd.yml logs --tail=100 qkd-api
```

### Grafana Dashboards

1. Open http://localhost:3000
2. Login with admin credentials from `.env.qkd`
3. Navigate to Dashboards → QKD → QKD System Overview

Key metrics to monitor:
- **Active Sessions**: Should be < 50 for optimal performance
- **QBER**: Should be < 11% (threshold for security)
- **API Response Time (p95)**: Should be < 500ms
- **Error Rate**: Should be < 1%

### Prometheus Queries

Access http://localhost:9091 and try:

```promql
# Sessions created per minute
rate(qkd_sessions_created_total[1m])

# Average QBER over last 5 minutes
avg_over_time(qkd_session_qber[5m])

# 95th percentile API response time
histogram_quantile(0.95, rate(qkd_http_request_duration_seconds_bucket[5m]))

# Current active sessions
qkd_sessions_active
```

### Alerts

Configure alerts in Grafana or Prometheus for:
- High QBER (> 11%)
- High error rate (> 5%)
- Low available connections
- High memory usage

## Production Deployment

### Security Hardening

1. **Change all default passwords** in `.env.qkd`

2. **Enable TLS/HTTPS:**
   ```env
   TLS_ENABLED=true
   TLS_CERT_FILE=/certs/server.crt
   TLS_KEY_FILE=/certs/server.key
   ```

3. **Use strong JWT secret** (min 32 characters)

4. **Enable PostgreSQL SSL:**
   ```env
   DB_SSL_MODE=require
   ```

5. **Restrict network access:**
   - Use firewall rules
   - Configure Docker network policies
   - Use reverse proxy (nginx/traefik)

### Backup Strategy

**PostgreSQL:**
```bash
# Backup
docker exec qkd-postgres pg_dump -U qkd_user qkd_production > backup.sql

# Restore
docker exec -i qkd-postgres psql -U qkd_user qkd_production < backup.sql
```

**Redis:**
```bash
# Backup (RDB snapshot)
docker exec qkd-redis redis-cli -a $REDIS_PASSWORD SAVE

# Copy backup file
docker cp qkd-redis:/data/dump.rdb ./redis-backup.rdb
```

### Scaling

**Horizontal scaling:**
```yaml
# In docker-compose.qkd.yml
services:
  qkd-api:
    deploy:
      replicas: 3
      mode: replicated
```

**Resource limits:**
Adjust CPU/memory limits in `docker-compose.qkd.yml`:
```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 2G
```

### Health Checks

**API Health:**
```bash
curl http://localhost:8080/health
```

**Expected response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-18T12:00:00Z",
  "services": {
    "database": "healthy",
    "cache": "healthy",
    "quantum_backend": "healthy"
  }
}
```

## Troubleshooting

### Services Won't Start

**Check logs:**
```bash
docker-compose -f docker-compose.qkd.yml logs
```

**Common issues:**
- Port conflicts: Change ports in docker-compose.qkd.yml
- Missing .env.qkd: Copy from .env.qkd.example
- Insufficient resources: Check Docker daemon settings

### Database Connection Errors

```bash
# Check PostgreSQL is running
docker-compose -f docker-compose.qkd.yml ps postgres

# View PostgreSQL logs
docker-compose -f docker-compose.qkd.yml logs postgres

# Test connection
docker exec qkd-postgres psql -U qkd_user -d qkd_production -c "SELECT version();"
```

### High QBER

If QBER consistently exceeds 11%:
- Check quantum backend status
- Verify channel noise settings
- Review session creation parameters
- Check for potential eavesdropping (in production)

### Memory Issues

```bash
# Check memory usage
docker stats

# Increase Redis max memory
# Edit docker-compose.qkd.yml:
command: >
  redis-server
  --maxmemory 512mb  # Increase this value
```

### Performance Issues

**Database:**
```sql
-- Check slow queries
SELECT * FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;

-- Check indexes
SELECT * FROM pg_stat_user_indexes WHERE idx_scan = 0;
```

**API:**
```bash
# Check concurrent connections
curl http://localhost:8080/metrics | grep qkd_http_requests_total

# View response times
curl http://localhost:8080/metrics | grep qkd_http_request_duration
```

### Cleanup

**Stop services:**
```bash
docker-compose -f docker-compose.qkd.yml down
```

**Remove volumes (WARNING: Deletes all data):**
```bash
docker-compose -f docker-compose.qkd.yml down -v
```

**Prune unused resources:**
```bash
docker system prune -a
```

## Support

For issues, feature requests, or questions:
- GitHub Issues: https://github.com/jaskrrish/Go-OKD/issues
- Documentation: See `/docs` directory

## License

See LICENSE file in repository root.
