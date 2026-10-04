# Dunder Mifflin Inventory System

A secure inventory management web application themed on *The Office*, built for ENPM680 (Fall 2025). The project walks through requirements, design, threat modeling, and a secure implementation.

**Stack:** Go · [Fiber](https://gofiber.io/) · [GORM](https://gorm.io/) · PostgreSQL · Bootstrap/jQuery frontend

## Features

- Product management: add, view, search/filter, and update stock with optimistic locking
- Low stock alerts with CRITICAL/WARNING severity and acknowledgment
- Audit logging of all actions (user, action, old/new values, IP, timestamp)
- Role-based access control: Admin, Manager, Employee, Associate
- Security controls: bcrypt password hashing (cost 12), password strength policy, input validation, security event logging

## Project Structure

```
.
├── main.go               # Application server (routes, models, auth, handlers)
├── main_test.go          # Unit and integration tests
├── go.mod / go.sum       # Go module definition
├── db/
│   └── schema.sql        # PostgreSQL schema
├── public/               # Static frontend (HTML, CSS, JS, images, vendor libs)
└── docs/
    ├── QUICKSTART.md
    ├── IMPLEMENTATION_GUIDE.md
    ├── IMPLEMENTATION_SUMMARY.md
    ├── design/           # Use cases, requirements and design specifications
    └── guides/           # Build/setup guide and user guide
```

## Getting Started

### Prerequisites

- Go 1.23+
- PostgreSQL 14+

### Setup

```bash
# 1. Create the database
createdb -U postgres dundermifflin

# 2. (Optional) point the app at your database; defaults to localhost postgres/postgres
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=dundermifflin port=5432 sslmode=disable"

# 3. Install dependencies and run
go mod download
go run .
```

The server starts on http://localhost:3000. Tables are created automatically on startup; `db/schema.sql` is provided for reference or manual setup.

### Run tests

```bash
go test ./...
```

## Demo Accounts

Seeded on startup for local testing only. Change or remove them before any real deployment.

| Username | Role      | Password          |
|----------|-----------|-------------------|
| admin    | Admin     | `Admin123!@#$`    |
| mscott   | Manager   | `Manager123!@#$`  |
| dwight   | Employee  | `Employee123!@#$` |
| jim      | Associate | `Sales123!@#$`    |

## Documentation

- [Quick Start](docs/QUICKSTART.md)
- [Build & Setup Guide](docs/guides/BUILD_SETUP_GUIDE.md)
- [User Guide](docs/guides/USER_GUIDE.md)
- [Implementation Guide](docs/IMPLEMENTATION_GUIDE.md)
- [Design documents](docs/design/)

## License

[MIT](LICENSE)
