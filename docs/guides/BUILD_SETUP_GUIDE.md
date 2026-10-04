# Dunder Mifflin Inventory System
## Build and Setup Guide

**Student Name:** Amogh  
**UMD Directory ID:** amoghdp  
**Date:** October 24, 2025  
**Version:** 1.0

---

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Dependencies](#dependencies)
3. [Database Setup](#database-setup)
4. [Configuration](#configuration)
5. [Build Process](#build-process)
6. [Running the Application](#running-the-application)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)

---

## 1. Prerequisites

### Required Software

**Go Programming Language (version 1.21 or higher)**
- Download from: https://go.dev/dl/
- Installation:
  - macOS: `brew install go`
  - Linux: `sudo apt install golang-go`
  - Windows: Download and run installer
- Verify: `go version`

**PostgreSQL Database (version 13 or higher)**
- Download from: https://www.postgresql.org/download/
- Installation:
  - macOS: `brew install postgresql@15`
  - Linux: `sudo apt install postgresql postgresql-contrib`
  - Windows: Download and run installer
- Verify: `psql --version`

**Git (for version control)**
- Download from: https://git-scm.com/downloads
- Verify: `git --version`

### System Requirements
- Operating System: Windows 10/11, macOS 10.15+, or Linux
- RAM: Minimum 4GB (8GB recommended)
- Disk Space: At least 500MB free space
- Internet connection (for downloading dependencies)

---

## 2. Dependencies

All Go dependencies are managed through Go modules (`go.mod` file).

### Main Dependencies:
- **Fiber v2.52.0** - Web framework
- **GORM v1.25.5** - ORM for database operations
- **PostgreSQL Driver v1.5.4** - Database connector
- **Testify v1.8.4** - Testing framework

### Installing Dependencies

```bash
# Navigate to project directory
cd ENPM680Fall25Project-amoghdp

# Download all dependencies
go mod download

# Verify dependencies
go mod verify
```

Expected output:
```
all modules verified
```

---

## 3. Database Setup

### Step 1: Start PostgreSQL Service

**macOS:**
```bash
brew services start postgresql@15
brew services list | grep postgresql
```

**Linux:**
```bash
sudo systemctl start postgresql
sudo systemctl status postgresql
```

**Windows:**
- Open Services app
- Find "PostgreSQL" service
- Click "Start"

### Step 2: Create PostgreSQL User

```bash
# Create superuser named 'postgres' (if it doesn't exist)
createuser -s postgres

# Or use your system username
whoami  # Check your username
```

### Step 3: Create Database

```bash
# Create the database
createdb dundermifflin

# Verify database was created
psql -l | grep dundermifflin
```

### Step 4: Test Database Connection

```bash
psql -U postgres -d dundermifflin -c "SELECT 1"
```

Expected output:
```
 ?column? 
----------
        1
(1 row)
```

### Database Schema

The application automatically creates tables on first run using GORM auto-migration:

**Products Table:**
```sql
CREATE TABLE products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

**Users Table:**
```sql
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR UNIQUE NOT NULL,
    role VARCHAR,
    password_hash VARCHAR
);
```

---

## 4. Configuration

### Database Connection

The database connection is configured in `main.go`:

```go
dsn := "host=localhost user=postgres dbname=dundermifflin port=5432 sslmode=disable"
```

**If your PostgreSQL setup is different, update the connection string:**
- `host`: Database server hostname (default: localhost)
- `user`: PostgreSQL username (default: postgres)
- `dbname`: Database name (dundermifflin)
- `port`: PostgreSQL port (default: 5432)
- `sslmode`: SSL mode (disable for local development)

### Server Configuration

The server runs on port 3000 by default. To change:

Edit `main.go`, line ~210:
```go
log.Fatal(app.Listen(":3000"))  // Change :3000 to your port
```

### Static Files

Frontend files are served from the `public/` directory:
```
public/
├── index.html
├── login.html
├── inventory.html
├── css/
├── js/
├── img/
└── lib/
```

---

## 5. Build Process

### Step 1: Clone Repository

```bash
git clone https://code.umd.edu/<username>/ENPM680Fall25Project-amoghdp.git
cd ENPM680Fall25Project-amoghdp
```

### Step 2: Checkout Phase3 Branch

```bash
git checkout Phase3
```

### Step 3: Install Dependencies

```bash
go mod download
go mod tidy
```

### Step 4: Build the Application

```bash
# Build executable
go build -o dundermifflin main.go

# On Windows, output will be dundermifflin.exe
# On macOS/Linux, output will be dundermifflin
```

Verify build:
```bash
ls -la dundermifflin*  # macOS/Linux
dir dundermifflin.exe  # Windows
```

---

## 6. Running the Application

### Option 1: Run Directly (Development)

```bash
go run main.go
```

### Option 2: Run Compiled Binary (Production)

```bash
# macOS/Linux
./dundermifflin

# Windows
dundermifflin.exe
```

### Expected Output

```
✓ Database connected and migrated successfully (PostgreSQL)
✓ Server starting on http://localhost:3000

 ┌───────────────────────────────────────────────────┐ 
 │          Fiber v2.52.0                            │ 
 │          http://127.0.0.1:3000                    │ 
 │                                                   │ 
 │ Handlers ............ 7  Processes ........... 1  │ 
 │ Prefork ....... Disabled  PID ............. 12345 │ 
 └───────────────────────────────────────────────────┘
```

---

## 7. Verification

### Step 1: Check Health Endpoint

```bash
curl http://localhost:3000/api/v1/health
```

Expected response:
```json
{
  "status": "ok",
  "message": "Dunder Mifflin Inventory System is running"
}
```

### Step 2: Access Frontend

Open browser and navigate to:
```
http://localhost:3000
```

You should see the Dunder Mifflin homepage.

### Step 3: Test Login

1. Navigate to: `http://localhost:3000/login.html`
2. Login with: `admin` / `admin123`
3. Should redirect to inventory page

### Step 4: Test API Endpoints

**Add a product:**
```bash
curl -X POST http://localhost:3000/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Copy Paper A4",
    "description": "Premium white paper",
    "quantity": 1000
  }'
```

**Get all products:**
```bash
curl http://localhost:3000/api/v1/products
```

### Step 5: Run Tests

```bash
go test -v
```

Expected output:
```
=== RUN   TestAddProduct_Success
--- PASS: TestAddProduct_Success (0.00s)
=== RUN   TestAddProduct_Duplicate
--- PASS: TestAddProduct_Duplicate (0.00s)
...
PASS
ok      dundermifflin    0.382s
```

---

## 8. Troubleshooting

### Issue 1: "Failed to connect to database"

**Symptoms:**
```
Failed to connect to database: dial tcp [::1]:5432: connect: connection refused
```

**Solutions:**
1. Check if PostgreSQL is running:
   ```bash
   # macOS
   brew services list | grep postgresql
   
   # Linux
   sudo systemctl status postgresql
   ```

2. Start PostgreSQL if stopped:
   ```bash
   # macOS
   brew services start postgresql@15
   
   # Linux
   sudo systemctl start postgresql
   ```

3. Verify database exists:
   ```bash
   psql -l | grep dundermifflin
   ```

### Issue 2: "Port 3000 already in use"

**Symptoms:**
```
listen tcp :3000: bind: address already in use
```

**Solutions:**
1. Find process using port 3000:
   ```bash
   # macOS/Linux
   lsof -i :3000
   
   # Windows
   netstat -ano | findstr :3000
   ```

2. Kill the process or change port in `main.go`

### Issue 3: "role postgres does not exist"

**Symptoms:**
```
FATAL: role "postgres" does not exist
```

**Solutions:**
1. Create postgres user:
   ```bash
   createuser -s postgres
   ```

2. Or use your system username in `main.go`:
   ```go
   dsn := "host=localhost user=<your-username> dbname=dundermifflin ..."
   ```

### Issue 4: "cannot find module"

**Symptoms:**
```
go: cannot find module providing package github.com/gofiber/fiber/v2
```

**Solutions:**
```bash
go mod download
go mod tidy
```

### Issue 5: Tests Failing

**Solutions:**
1. Ensure database is running
2. Run tests individually:
   ```bash
   go test -v -run TestAddProduct_Success
   ```
3. Check `main_test.go` for syntax errors

---

## Additional Resources

- **Go Documentation**: https://go.dev/doc/
- **Fiber Framework**: https://docs.gofiber.io/
- **GORM ORM**: https://gorm.io/docs/
- **PostgreSQL Docs**: https://www.postgresql.org/docs/

---

## Support

For issues or questions:
- **Email**: amoghdp@umd.edu
- **Repository**: https://code.umd.edu/<username>/ENPM680Fall25Project-amoghdp

---

**Document Version:** 1.0  
**Last Updated:** October 24, 2025
