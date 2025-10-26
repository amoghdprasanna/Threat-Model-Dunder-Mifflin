# Quick Start Guide - Dunder Mifflin Inventory System

## Prerequisites
- Go 1.21 or higher
- PostgreSQL 14 or higher
- Basic terminal/command line knowledge

## 5-Minute Setup

### Step 1: Install PostgreSQL (if not already installed)

**macOS:**
```bash
brew install postgresql@14
brew services start postgresql@14
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

**Windows:**
Download and install from https://www.postgresql.org/download/windows/

### Step 2: Create Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database (in psql prompt)
CREATE DATABASE dundermifflin;

# Exit
\q
```

Or use this one-liner:
```bash
createdb -U postgres dundermifflin
```

### Step 3: Set Up Project

```bash
# Create project directory
mkdir dundermifflin-inventory
cd dundermifflin-inventory

# Copy the provided files:
# - main.go
# - go.mod

# Download dependencies
go mod download
```

### Step 4: Configure Database Connection (Optional)

If your PostgreSQL has a password or uses different settings:

```bash
# Set environment variable
export DATABASE_URL="host=localhost user=postgres password=yourpassword dbname=dundermifflin port=5432 sslmode=disable"
```

Or edit the DSN in `main.go` line 151:
```go
dsn := "host=localhost user=postgres password=yourpassword dbname=dundermifflin port=5432 sslmode=disable"
```

### Step 5: Run the Application

```bash
go run main.go
```

You should see:
```
✓ Database connected and migrated successfully (PostgreSQL)
✓ Created user: admin (password: Admin123!)
✓ Created user: mscott (password: Manager123!)
✓ Created user: dwight (password: Employee123!)
✓ Created user: jim (password: Sales123!)
✓ Server starting on http://localhost:3000
```

## Quick Test

### Test 1: Health Check
```bash
curl http://localhost:3000/api/v1/health
```

Expected response:
```json
{
  "status": "healthy",
  "message": "Dunder Mifflin Inventory System is running",
  "timestamp": "2025-10-25T..."
}
```

### Test 2: Login
```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!"
  }' \
  -c cookies.txt
```

Expected response:
```json
{
  "session_id": "...",
  "user": {
    "user_id": 1,
    "username": "admin",
    "role": "Administrator",
    ...
  },
  "expires_at": "..."
}
```

### Test 3: Create a Product
```bash
curl -X POST http://localhost:3000/api/v1/products \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "sku": "PAPER-001",
    "name": "Premium Copy Paper",
    "description": "20lb white, 500 sheets per ream",
    "category": "Office Paper",
    "quantity": 100,
    "low_stock_threshold": 20,
    "price": 8.99
  }'
```

Expected response:
```json
{
  "message": "Product created successfully",
  "product": {
    "product_id": 1,
    "sku": "PAPER-001",
    "name": "Premium Copy Paper",
    ...
  }
}
```

### Test 4: View Products
```bash
curl http://localhost:3000/api/v1/products \
  -b cookies.txt
```

Expected response:
```json
{
  "products": [
    {
      "product_id": 1,
      "sku": "PAPER-001",
      "name": "Premium Copy Paper",
      ...
    }
  ],
  "count": 1
}
```

### Test 5: Update Product Quantity
```bash
curl -X PUT http://localhost:3000/api/v1/products/1 \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "quantity": 15,
    "version": 1
  }'
```

This should trigger a low stock alert since 15 < 20 (threshold)!

### Test 6: View Low Stock Alerts (Manager/Admin only)
```bash
curl http://localhost:3000/api/v1/alerts \
  -b cookies.txt
```

Expected response:
```json
{
  "alerts": [
    {
      "alert_id": 1,
      "product_id": 1,
      "product_name": "Premium Copy Paper",
      "current_qty": 15,
      "threshold": 20,
      "severity": "WARNING",
      ...
    }
  ],
  "count": 1
}
```

## Default User Accounts

| Username | Password | Role |
|----------|----------|------|
| admin | Admin123! | Administrator |
| mscott | Manager123! | Warehouse Manager |
| dwight | Employee123! | Warehouse Employee |
| jim | Sales123! | Sales Associate |

## Common Issues & Solutions

### Issue: "Failed to connect to database"
**Solution:** 
1. Make sure PostgreSQL is running: `sudo systemctl status postgresql` or `brew services list`
2. Check if database exists: `psql -U postgres -l`
3. Verify connection settings in code or environment variable

### Issue: "port 3000 already in use"
**Solution:** 
1. Find process using port: `lsof -i :3000` (macOS/Linux) or `netstat -ano | findstr :3000` (Windows)
2. Kill the process or change port in `main.go` (line 821)

### Issue: "dial tcp: connection refused"
**Solution:**
PostgreSQL might not be accepting connections. Edit `/etc/postgresql/14/main/pg_hba.conf` and ensure:
```
host    all             all             127.0.0.1/32            md5
```

### Issue: "invalid or expired session"
**Solution:**
Sessions expire after 8 hours. Simply log in again:
```bash
curl -X POST http://localhost:3000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "Admin123!"}' \
  -c cookies.txt
```

## Next Steps

1. **Explore the API:** See README.md for complete API documentation
2. **Test Authorization:** Try logging in with different users (jim, dwight, mscott)
3. **Test Error Handling:** 
   - Try creating duplicate products
   - Try updating with wrong version number
   - Try deleting as Sales Associate (should fail)
4. **View Audit Logs:** Login as admin and check `/api/v1/audit-logs`
5. **Build Frontend:** Create a UI to interact with the API

## Using with Postman

1. Import as OpenAPI/Postman collection
2. Set `base_url` variable to `http://localhost:3000`
3. Login to get session_id
4. Use session_id in Cookie header for subsequent requests

## Using with Frontend

Example with fetch API:
```javascript
// Login
const login = async () => {
  const response = await fetch('http://localhost:3000/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include', // Important: includes cookies
    body: JSON.stringify({
      username: 'admin',
      password: 'Admin123!'
    })
  });
  return await response.json();
};

// Get products
const getProducts = async () => {
  const response = await fetch('http://localhost:3000/api/v1/products', {
    credentials: 'include' // Important: sends cookies
  });
  return await response.json();
};
```

## Production Deployment

Before deploying to production:

1. **Environment Variables:**
   ```bash
   export DATABASE_URL="your-production-database-url"
   export PORT="3000"
   ```

2. **Enable HTTPS:**
   - Change `Secure: false` to `Secure: true` in cookie settings (line 638)
   - Use reverse proxy (nginx) with SSL certificate

3. **Security Hardening:**
   - Change default passwords
   - Enable CORS only for your domain
   - Add rate limiting
   - Enable logging to file
   - Set up monitoring

4. **Database:**
   - Use connection pooling
   - Enable SSL for database connection
   - Regular backups
   - Set up read replicas if needed

## Getting Help

- Check README.md for detailed documentation
- Review IMPLEMENTATION_SUMMARY.md for features overview
- Check logs for error messages
- Review code comments in main.go

## Success Checklist

- [ ] PostgreSQL installed and running
- [ ] Database `dundermifflin` created
- [ ] Application starts without errors
- [ ] Health check returns "healthy"
- [ ] Can login with admin user
- [ ] Can create a product
- [ ] Can view products
- [ ] Can update product quantity
- [ ] Low stock alert appears when quantity is low
- [ ] Can view audit logs as admin

If all items are checked, you're ready to go! 🎉

---

**Need More Help?** 
- Review the full README.md
- Check the Go documentation: https://go.dev/doc/
- Check Fiber documentation: https://docs.gofiber.io/
- Check GORM documentation: https://gorm.io/docs/
