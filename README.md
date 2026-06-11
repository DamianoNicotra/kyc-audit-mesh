# KYC Audit Mesh

**Immutable audit trail for KYC events with SHA256 hash chain. Built with Go, Azure, and Snowflake.**

[![Go Version](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org/)
[![Azure](https://img.shields.io/badge/Azure-Container%20Apps-0089D6.svg)](https://azure.microsoft.com/)
[![Snowflake](https://img.shields.io/badge/Snowflake-Audit%20Logs-29B5E8.svg)](https://www.snowflake.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 🚀 Overview

KYC Audit Mesh is a production-ready audit logging system that creates an **immutable, tamper-evident chain of events** for KYC (Know Your Customer) processes. Every event is cryptographically linked to the previous one using SHA256 hashes.

**Use cases:** AML compliance, financial audits, regulatory reporting, identity verification trails.

## 🏗️ Architecture
Client → KYC Ingestor (Go) → Snowflake (Audit Table)
↓
Hash chain calculation (SHA256)
↓
In-memory cache + Async persistence


## 📦 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/kyc/event` | Submit a KYC event (verification, doc upload, approval, etc.) |
| `GET` | `/kyc/events` | Retrieve all events from in-memory cache |
| `GET` | `/health` | Health check for the service |

### Example: Submit a KYC Event

```bash
curl -X POST http://localhost:8080/kyc/event \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "action": "verification",
    "details": {"status": "approved", "reviewer": "auto"},
    "ip_address": "192.168.1.100"
  }'
