# Convo

Convo is a Go-based backend project designed for real-time communication, featuring modular architecture and support for WebSocket connections. It is structured for scalability and maintainability, making it suitable for chat applications, collaborative platforms, and more.

## Overview

- **Language:** Go
- **Main Entry Point:** `cmd/api/main.go`
- **WebSocket Support:** Integrated via Gorilla WebSocket
- **Modular Structure:**
  - `internal/handlers/` — HTTP and WebSocket handlers
  - `internal/models/` — Data models
  - `internal/server/` — Server setup and configuration
  - `internal/utils/` — Utility functions
  - `internal/database/` — Database integration
  - `migrations/` — SQL migration scripts
  - `cpp/` — C++ integrations (optional)

## Features

- User authentication and signup
- Room creation and management
- Real-time messaging
- Configurable middleware (logging, token validation)
- Database migrations

---

*Continue below for setup instructions, usage, and API documentation.*
