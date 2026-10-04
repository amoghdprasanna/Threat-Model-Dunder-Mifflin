# Dunder Mifflin Inventory System
## User Guide

**Student Name:** Amogh  
**UMD Directory ID:** amoghdp  
**Date:** October 24, 2025  
**Version:** 1.0

---

## Table of Contents
1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Logging In](#logging-in)
4. [Main Features](#main-features)
5. [Use Case Walkthroughs](#use-case-walkthroughs)
6. [Troubleshooting](#troubleshooting)

---

## 1. Introduction

The Dunder Mifflin Inventory System is a web-based application for managing paper product inventory. This system allows employees to:
- Add new products to inventory
- View current stock levels
- Update product quantities
- Search for products
- Delete products from inventory

### System Requirements
- Modern web browser (Chrome, Firefox, Safari, Edge)
- Internet connection
- Employee credentials for login

---

## 2. Getting Started

### Accessing the Application

1. Open your web browser
2. Navigate to: `http://localhost:3000`
3. You will see the Dunder Mifflin homepage

![Homepage Screenshot - Insert screenshot here]

### Navigation Menu

The main navigation menu includes:
- **Home** - Company homepage
- **About Us** - Company information
- **Services** - Our services
- **Our Team** - Team members
- **Employee Portal** - Login to inventory system
- **Contact** - Contact information

---

## 3. Logging In

### Step 1: Access Login Page

1. From the homepage, click **"Employee Portal"** in the navigation menu
2. You will be directed to the login page

![Login Page Screenshot - Insert screenshot here]

### Step 2: Enter Credentials

The system provides demo credentials for testing:

| Username | Password | Role |
|----------|----------|------|
| admin | admin123 | Administrator |
| mscott | WorldsBestBoss | Regional Manager |
| dwight | BeetFarmer123 | Assistant Regional Manager |
| jim | BigTuna | Sales Representative |
| pam | Receptionist | Receptionist |
| ryan | TempLife | Temp |

1. Enter your **username** in the first field
2. Enter your **password** in the second field
3. Click the **"Login"** button

### Step 3: Successful Login

Upon successful login:
- You will see a success message
- The system will redirect you to the Inventory Management page
- Your username will appear in the navigation bar

![Successful Login Screenshot - Insert screenshot here]

### Login Error Handling

If you enter incorrect credentials:
- An error message will appear: "Invalid username or password"
- The password field will be cleared
- Try again with correct credentials

---

## 4. Main Features

### 4.1 Inventory Dashboard

After logging in, you'll see the Inventory Management dashboard with:

**Statistics Cards:**
- **Total Products** - Number of unique products
- **Total Units in Stock** - Sum of all quantities

**Search Bar:**
- Search for products by name
- Results filter in real-time as you type

**Add Product Button:**
- Opens a modal to add new products

**Inventory Table:**
- Lists all products with details:
  - ID
  - Product Name
  - Description
  - Quantity
  - Status (In Stock, Low Stock, Out of Stock)
  - Last Updated timestamp
  - Action buttons (Update, Delete)

![Inventory Dashboard Screenshot - Insert screenshot here]

### 4.2 Product Status Indicators

The system uses color-coded badges to show stock status:

- **🟢 In Stock** (Green) - Quantity ≥ 100 units
- **🟡 Low Stock** (Yellow) - Quantity < 100 units
- **🔴 Out of Stock** (Red) - Quantity = 0 units

---

## 5. Use Case Walkthroughs

### Use Case 1: Adding a New Product

**Objective:** Add a new paper product to the inventory

**Steps:**

1. Click the **"Add Product"** button (green button at top right)

![Add Product Button Screenshot - Insert screenshot here]

2. A modal window will appear with a form

3. Fill in the product details:
   - **Product Name*** (Required) - e.g., "Copy Paper A4"
   - **Description** (Optional) - e.g., "Premium white copy paper"
   - **Quantity*** (Required) - e.g., 1000

![Add Product Modal Screenshot - Insert screenshot here]

4. Click **"Add Product"** button

5. Success message appears: "Product added successfully!"

6. The new product appears in the inventory table

7. Statistics update automatically

**Expected Result:**
- Product is added to database
- Appears in inventory list
- Total products count increases
- Total stock count increases

**Error Cases:**
- If product already exists: "Product already exists" error message
- If required fields are empty: Validation error

---

### Use Case 2: Viewing Current Stock Levels

**Objective:** View all products and their quantities

**Steps:**

1. After logging in, you automatically see the inventory table

2. The table displays:
   - All products currently in inventory
   - Each product's details
   - Stock status for each product

![Inventory Table Screenshot - Insert screenshot here]

3. **Using Search:**
   - Type in the search box at the top
   - Results filter automatically
   - Example: Type "paper" to see only paper products

![Search Functionality Screenshot - Insert screenshot here]

4. **Statistics:**
   - View total number of products
   - View total units across all products

**Expected Result:**
- All products are visible
- Stock levels are current
- Search works in real-time
- Statistics are accurate

---

### Use Case 3: Updating Product Quantity

**Objective:** Update the quantity of an existing product

**Steps:**

1. Find the product you want to update in the inventory table

2. Click the **"Update"** button (blue button) for that product

![Update Button Screenshot - Insert screenshot here]

3. A modal window appears showing:
   - Product Name (read-only)
   - Current Quantity (read-only)
   - New Quantity field (editable)

![Update Product Modal Screenshot - Insert screenshot here]

4. Enter the new quantity
   - Example: Change from 1000 to 1500

5. Click **"Update"** button

6. Success message appears: "Product updated successfully!"

7. The product row updates with:
   - New quantity
   - Updated timestamp
   - Possibly new status badge

8. Total stock statistic updates automatically

**Expected Result:**
- Product quantity is updated
- Last Updated timestamp changes
- Status badge may change based on new quantity
- Total stock count reflects change

**Error Cases:**
- Negative quantity: "Quantity cannot be negative" error
- Non-existent product: "Product not found" error

---

### Use Case 4: Searching for Products

**Objective:** Find specific products quickly

**Steps:**

1. Locate the search box at the top of the inventory page

2. Type part of a product name
   - Example: "copy" to find "Copy Paper A4"
   - Search is case-insensitive

3. Results filter automatically as you type

4. Products matching your search appear in the table

5. Clear the search box to show all products again

![Search Results Screenshot - Insert screenshot here]

**Expected Result:**
- Only matching products are shown
- Statistics update for filtered results
- Clearing search shows all products

---

### Use Case 5: Deleting a Product

**Objective:** Remove a product from inventory

**Steps:**

1. Find the product you want to delete

2. Click the **"Delete"** button (red button) for that product

![Delete Button Screenshot - Insert screenshot here]

3. A confirmation dialog appears:
   "Delete this product?"

4. Click **"OK"** to confirm, or **"Cancel"** to abort

5. If confirmed:
   - Success message: "Product deleted successfully!"
   - Product disappears from table
   - Statistics update automatically

**Expected Result:**
- Product is removed from database
- No longer appears in inventory
- Total products count decreases
- Total stock count decreases

**Warning:** This action cannot be undone!

---

### Use Case 6: Logging Out

**Objective:** Safely exit the system

**Steps:**

1. Click **"Logout (Your Name)"** in the navigation bar

![Logout Link Screenshot - Insert screenshot here]

2. A confirmation dialog appears:
   "Are you sure you want to logout?"

3. Click **"OK"** to logout

4. You are redirected to the login page

5. Your session is cleared

**Expected Result:**
- Session is terminated
- Redirected to login page
- Cannot access inventory without logging in again

---

## 6. Troubleshooting

### Problem 1: Cannot Login

**Symptoms:**
- "Invalid username or password" message appears
- Cannot access inventory system

**Solutions:**
1. Verify you're using correct credentials
2. Try demo account: `admin` / `admin123`
3. Check if Caps Lock is on
4. Clear browser cache and try again

---

### Problem 2: Inventory Not Loading

**Symptoms:**
- "Error loading inventory" message
- Blank inventory table
- Spinning loader doesn't stop

**Solutions:**
1. Refresh the page (F5 or Ctrl+R)
2. Check internet connection
3. Verify server is running (check with administrator)
4. Clear browser cache
5. Try different browser

---

### Problem 3: Cannot Add Product

**Symptoms:**
- "Product already exists" error
- Form doesn't submit
- No success message

**Solutions:**
1. **If "Product already exists":**
   - Choose a different product name
   - Or update the existing product instead

2. **If form won't submit:**
   - Ensure all required fields (*) are filled
   - Check quantity is a positive number
   - Try refreshing page

---

### Problem 4: Search Not Working

**Symptoms:**
- Typing in search box shows no results
- Search box doesn't respond

**Solutions:**
1. Refresh the page
2. Clear the search box completely
3. Try different search terms
4. Check if products exist with that name

---

### Problem 5: Update Fails

**Symptoms:**
- "Failed to update product" error
- Quantity doesn't change

**Solutions:**
1. Ensure new quantity is not negative
2. Check you're entering a valid number
3. Refresh page and try again
4. Verify you have permission to update

---

### Problem 6: Unexpected Logout

**Symptoms:**
- Suddenly redirected to login page
- Session ends unexpectedly

**Solutions:**
1. This is normal after period of inactivity
2. Simply login again
3. Your data is saved automatically

---

## Quick Reference

### Keyboard Shortcuts

- **Enter** - Submit login form
- **Escape** - Close modal dialogs
- **Ctrl/Cmd + R** - Refresh page

### Status Badge Colors

- 🟢 **Green** = In Stock (≥100 units)
- 🟡 **Yellow** = Low Stock (<100 units)  
- 🔴 **Red** = Out of Stock (0 units)

### Demo Credentials

Quick reference for testing:
```
admin / admin123
mscott / WorldsBestBoss
dwight / BeetFarmer123
```

---

## Support and Feedback

### Need Help?

- **Technical Issues**: Contact IT support
- **Feature Requests**: Submit via email
- **Bug Reports**: Report to development team

### Contact Information

- **Email**: amoghdp@umd.edu
- **System Administrator**: [Your Name]
- **Help Desk**: Available during business hours

---

## Appendix: Screenshots

**Note:** This section should include the following screenshots:

1. Homepage
2. Login page (empty)
3. Login page (with credentials)
4. Successful login message
5. Inventory dashboard (with data)
6. Add product modal
7. Product added successfully
8. Search functionality
9. Update product modal
10. Product updated successfully
11. Delete confirmation dialog
12. Logout confirmation

**To add screenshots:**
1. Take screenshots using your system's screenshot tool
2. Insert them in the marked locations above
3. Add captions describing what each screenshot shows

---

**Document Version:** 1.0  
**Last Updated:** October 24, 2025  
**Next Review Date:** November 24, 2025
