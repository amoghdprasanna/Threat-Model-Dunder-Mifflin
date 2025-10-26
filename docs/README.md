# Dunder Mifflin Inventory System - Implementation Guide

## Overview
This is a comprehensive inventory management system with full authentication, authorization, audit logging, and low stock alerts - implementing all functional and non-functional requirements.

## What's Been Implemented

### ✅ Functional Requirements

#### FR-001: Add New Product
- Full validation (name length, SKU uniqueness, required fields)
- Duplicate detection (SKU and name)
- Comprehensive error handling
- Audit trail logging

#### FR-002: View Inventory
- Product listing with search/filter
- Category filtering
- Low stock filtering
- Detailed product views

#### FR-003: Update Product Quantity
- Optimistic locking (concurrent modification detection)
- Version control
- Comprehensive validation
- Audit trail with old/new values

#### FR-004: Low Stock Alerts ⭐ NEW
- Automatic alert generation when quantity ≤ threshold
- Severity levels (CRITICAL/WARNING)
- Alert acknowledgment system
- Manager/Admin only access

#### FR-005: Audit Logging ⭐ NEW
- Comprehensive audit trail for all actions
- Tracks: user, action, entity, old/new values, timestamp, IP
- Success/failure logging
- Admin-only access
- Pagination support

### ✅ Non-Functional Requirements

#### NFR-001: Input Validation
- Name: max 200 characters
- Description: max 500 characters
- Quantity: non-negative, max 1,000,000
- SKU: required, unique
- Detailed validation error messages

#### NFR-002: Response Time
- Database connection pooling
- Indexed queries
- Optimized database schema
- Prepared statements

#### NFR-003: Authentication & Authorization ⭐ NEW
- Secure password hashing (bcrypt)
- Session-based authentication
- Role-based access control (RBAC)
- Account lockout after 5 failed attempts (15-minute lockout)
- Session management with expiration (8-hour sessions)
- IP address and user agent tracking

### 🎯 Additional Features

1. **Enhanced Security**
   - Password hashing with bcrypt (cost 12)
   - Session tokens with secure generation
   - Failed login tracking
   - Account lockout mechanism
   - Session cleanup (expired sessions auto-deleted)

2. **Error Handling**
   - Database connection failures
   - Query timeouts
   - Concurrent modification conflicts
   - Network errors
   - Session expiration
   - Validation failures
   - All error scenarios from use cases implemented

3. **Audit System**
   - Every action logged with:
     - User ID and username
     - Action type (CREATE, UPDATE, DELETE, LOGIN, LOGOUT)
     - Entity type and ID
     - Old and new values
     - IP address
     - Timestamp
     - Success/failure status

4. **Role-Based Access Control**
   - **Administrator**: Full access to all features + audit logs
   - **Warehouse Manager**: All product operations + low stock alerts
   - **Warehouse Employee**: Add, view, update products
   - **Sales Associate**: View products only

## Setup Instructions

### Prerequisites
```bash
# Install Go 1.21+
# Install PostgreSQL 14+
# Install Node.js (for frontend)
```

### Database Setup
```bash
# Create database
createdb dundermifflin

# Or using psql
psql -U postgres
CREATE DATABASE dundermifflin;
```

### Installation
```bash
# Install Go dependencies
go mod init dundermifflin
go get github.com/gofiber/fiber/v2
go get github.com/gofiber/fiber/v2/middleware/cors
go get github.com/gofiber/fiber/v2/middleware/logger
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get golang.org/x/crypto/bcrypt

# Run the application
go run main.go
```

### Environment Variables (Optional)
```bash
export DATABASE_URL="host=localhost user=postgres password=yourpassword dbname=dundermifflin port=5432 sslmode=disable"
```

## Default User Accounts

The system automatically creates these users on first run:

| Username | Password | Role | Email |
|----------|----------|------|-------|
| admin | Admin123! | Administrator | admin@dundermifflin.com |
| mscott | Manager123! | Warehouse Manager | michael@dundermifflin.com |
| dwight | Employee123! | Warehouse Employee | dwight@dundermifflin.com |
| jim | Sales123! | Sales Associate | jim@dundermifflin.com |

## API Documentation

### Authentication

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "Admin123!"
}

Response:
{
  "session_id": "...",
  "user": { ... },
  "expires_at": "2025-10-26T02:00:00Z"
}
```

#### Logout
```http
POST /api/v1/auth/logout
Cookie: session_id=...

Response:
{
  "message": "Logged out successfully"
}
```

#### Get Current User
```http
GET /api/v1/auth/me
Cookie: session_id=...

Response:
{
  "user_id": 1,
  "username": "admin",
  "role": "Administrator",
  ...
}
```

### Products

#### Create Product (Requires: Employee/Manager/Admin)
```http
POST /api/v1/products
Cookie: session_id=...
Content-Type: application/json

{
  "sku": "PAPER-001",
  "name": "Premium Copy Paper",
  "description": "20lb white, 500 sheets",
  "category": "Office Paper",
  "quantity": 100,
  "low_stock_threshold": 20,
  "price": 8.99
}

Response (201):
{
  "message": "Product created successfully",
  "product": { ... }
}

Error Responses:
- 400: Validation errors
- 401: Not authenticated
- 403: Insufficient permissions
- 409: Duplicate SKU or name
- 500: Database error
```

#### Get All Products
```http
GET /api/v1/products?search=paper&category=Office&low_stock=true
Cookie: session_id=...

Response:
{
  "products": [...],
  "count": 10
}
```

#### Get Single Product
```http
GET /api/v1/products/1
Cookie: session_id=...

Response:
{
  "product_id": 1,
  "sku": "PAPER-001",
  "name": "Premium Copy Paper",
  ...
}

Error Responses:
- 404: Product not found
```

#### Update Product (Requires: Employee/Manager/Admin)
```http
PUT /api/v1/products/1
Cookie: session_id=...
Content-Type: application/json

{
  "quantity": 150,
  "version": 1
}

Response:
{
  "message": "Product updated successfully",
  "product": { ... }
}

Error Responses:
- 400: Invalid quantity
- 404: Product not found
- 409: Concurrent modification (version mismatch)
- 500: Database error
```

#### Delete Product (Requires: Manager/Admin)
```http
DELETE /api/v1/products/1
Cookie: session_id=...

Response:
{
  "message": "Product deleted successfully"
}
```

### Low Stock Alerts (Requires: Manager/Admin)

#### Get Alerts
```http
GET /api/v1/alerts?acknowledged=false
Cookie: session_id=...

Response:
{
  "alerts": [
    {
      "alert_id": 1,
      "product_id": 5,
      "product_name": "Premium Cardstock",
      "current_qty": 8,
      "threshold": 10,
      "severity": "WARNING",
      "created_at": "2025-10-25T14:30:00Z",
      "acknowledged": false
    }
  ],
  "count": 1
}
```

#### Acknowledge Alert
```http
POST /api/v1/alerts/1/acknowledge
Cookie: session_id=...

Response:
{
  "message": "Alert acknowledged successfully",
  "alert": { ... }
}
```

### Audit Logs (Requires: Admin)

#### Get Audit Logs
```http
GET /api/v1/audit-logs?page=1&page_size=50&action=CREATE_PRODUCT
Cookie: session_id=...

Response:
{
  "logs": [
    {
      "log_id": 123,
      "user_id": 2,
      "username": "mscott",
      "action": "CREATE_PRODUCT",
      "entity_type": "Product",
      "entity_id": 5,
      "old_value": "",
      "new_value": "Premium Cardstock",
      "ip_address": "192.168.1.100",
      "timestamp": "2025-10-25T14:30:00Z",
      "success": true,
      "error_msg": ""
    }
  ],
  "page": 1,
  "page_size": 50,
  "total": 245,
  "total_pages": 5
}
```

### Health Check (Public)

```http
GET /api/v1/health

Response:
{
  "status": "healthy",
  "message": "Dunder Mifflin Inventory System is running",
  "timestamp": "2025-10-25T14:30:00Z"
}
```

## Error Handling

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| UNAUTHORIZED | 401 | No valid session |
| SESSION_EXPIRED | 401 | Session has expired |
| FORBIDDEN | 403 | Insufficient permissions |
| ACCOUNT_LOCKED | 403 | Too many failed login attempts |
| ACCOUNT_DISABLED | 403 | Account is disabled |
| NOT_FOUND | 404 | Resource not found |
| DUPLICATE_SKU | 409 | SKU already exists |
| DUPLICATE_NAME | 409 | Product name already exists |
| CONCURRENT_MODIFICATION | 409 | Product modified by another user |
| VALIDATION_ERROR | 400 | Input validation failed |
| INVALID_REQUEST | 400 | Malformed request |
| DATABASE_ERROR | 500 | Database operation failed |
| SESSION_ERROR | 500 | Failed to create session |

### Example Error Response
```json
{
  "error": "Product with this SKU already exists",
  "code": "DUPLICATE_SKU",
  "existing_product": {
    "product_id": 5,
    "sku": "PAPER-001",
    "name": "Premium Copy Paper"
  }
}
```

## Database Schema

### Tables Created

1. **products** - Product inventory
2. **users** - User accounts
3. **sessions** - Active user sessions
4. **audit_logs** - Comprehensive audit trail
5. **low_stock_alerts** - Low stock notifications

### Key Indexes

- `idx_products_quantity` - For low stock queries
- `idx_audit_logs_timestamp` - For audit log queries
- `idx_sessions_expires_at` - For session cleanup
- Username and email indexes on users
- SKU index on products

## Security Features

1. **Password Security**
   - bcrypt hashing with cost factor 12
   - Passwords never stored in plain text
   - Passwords never returned in API responses

2. **Session Management**
   - Cryptographically secure session IDs
   - 8-hour session duration
   - Automatic cleanup of expired sessions
   - Session tied to user agent (basic fingerprinting)

3. **Account Protection**
   - Failed login tracking
   - Automatic account lockout (5 attempts)
   - 15-minute lockout duration
   - Account status checking

4. **Authorization**
   - Role-based access control
   - Middleware enforcement
   - Unauthorized access logging

## Testing

### Manual Testing Checklist

#### Authentication
- [ ] Login with valid credentials
- [ ] Login with invalid credentials
- [ ] Account lockout after 5 failed attempts
- [ ] Session expiration after 8 hours
- [ ] Logout functionality

#### Product Management
- [ ] Create product with valid data
- [ ] Create product with duplicate SKU (should fail)
- [ ] Create product with missing required fields (should fail)
- [ ] Create product with negative quantity (should fail)
- [ ] View product list
- [ ] Search products by name
- [ ] Filter products by category
- [ ] Update product quantity
- [ ] Concurrent update detection
- [ ] Delete product

#### Low Stock Alerts
- [ ] Alert created when quantity ≤ threshold
- [ ] Alert severity (CRITICAL for qty=0, WARNING otherwise)
- [ ] View unacknowledged alerts
- [ ] Acknowledge alert

#### Audit Logging
- [ ] Login events logged
- [ ] Product creation logged
- [ ] Product updates logged with old/new values
- [ ] Product deletion logged
- [ ] Failed operations logged

#### Authorization
- [ ] Sales Associate cannot create products
- [ ] Employee cannot delete products
- [ ] Only Manager/Admin can view alerts
- [ ] Only Admin can view audit logs

## Performance Considerations

1. **Database Connection Pooling**
   - Max 100 connections
   - 10 idle connections
   - 1-hour connection lifetime

2. **Query Optimization**
   - Indexed columns for common queries
   - Prepared statements enabled
   - Selective field loading

3. **Session Management**
   - Hourly cleanup of expired sessions
   - Efficient session lookups with indexes

## Future Enhancements

1. **Frontend Implementation**
   - React/Vue.js UI
   - Real-time notifications
   - Dashboard with analytics

2. **Additional Features**
   - Export audit logs to CSV
   - Email notifications for alerts
   - Batch product import
   - Product categories management
   - Supplier management
   - Purchase order integration

3. **Testing**
   - Unit tests
   - Integration tests
   - API endpoint tests
   - Performance testing

## Troubleshooting

### Database Connection Issues
```
Error: Failed to connect to database
Solution: Check PostgreSQL is running and DATABASE_URL is correct
```

### Session Issues
```
Error: Invalid or expired session
Solution: Log in again to create a new session
```

### Permission Denied
```
Error: You do not have permission to perform this action
Solution: Check your user role and required permissions
```

## License

This is an educational project for the Dunder Mifflin Paper Company.

## Contact

For questions or issues, please contact the development team.
