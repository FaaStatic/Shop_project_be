# Shop POS Backend — Go Fiber v3

A point-of-sale (POS) backend for small shops, built with **Go Fiber v3** and Clean Architecture. Handles products, customers, transactions, debts (kasbon), and user sessions — with report export to Excel and PDF.

## Features

- **Domain modules**: products, customers, transactions, debt tracking, users/sessions
- **Report generation**: Excel export (excelize) and PDF (fpdf)
- **CLI tooling with Cobra**: run server, `migrate`, and `migrate:reset` as subcommands
- **Config via Viper** (env/file based)
- **Redis** caching/session layer
- **Swagger UI** served through `gofiber/contrib/v3/swaggerui`
- Request validation with go-playground/validator, JSON via bytedance/sonic

## Architecture

```
├── cmd/                 # Cobra commands (serve, migrate, migrate:reset)
├── config/              # Env & Fiber configuration
├── infrastructure/
│   ├── database/        # Postgres init + migrations (GORM)
│   ├── cache/           # Redis
│   └── logger/          # Custom logger
└── internal/
    ├── dto/             # Request/response DTOs per domain
    ├── repository/      # Data access layer
    ├── service/         # Business logic
    └── handler/         # Fiber v3 handlers (Ctx as value type)
```

## Stack

Go 1.26 · Fiber v3 · GORM + PostgreSQL · Redis · Cobra · Viper · Swagger UI · excelize · fpdf

## Running

```bash
# 1. Configure (see config/env_config)
# 2. Migrate
go run . migrate
# 3. Serve
go run .
# Swagger available at /swagger
```

## Notes on Fiber v3

This project targets Fiber **v3** (not v2): handlers receive `fiber.Ctx` as a value type, and contrib packages use the `/v3` module path — worth noting since most online examples still show v2 idioms.
