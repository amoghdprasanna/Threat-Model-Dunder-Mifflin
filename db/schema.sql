-- Dunder Mifflin Inventory System Database Schema
-- PostgreSQL

-- Products table
CREATE TABLE IF NOT EXISTS products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(100),
    password_hash VARCHAR(255)
);

-- Sample data (optional)
INSERT INTO products (name, description, quantity) VALUES
    ('Copy Paper A4', 'Premium white copy paper', 1000),
    ('Legal Pad', 'Yellow legal pads', 500),
    ('Stapler', 'Desktop stapler', 50)
ON CONFLICT DO NOTHING;
