# 🔥 sqleton MySQL Demo Examples 🔥

This document contains comprehensive examples showcasing sqleton's powerful features with a realistic e-commerce database.

## Setup

### 1. Start MySQL Container
```bash
docker run --name mysql-demo -e MYSQL_ROOT_PASSWORD=demo123 -e MYSQL_DATABASE=ecommerce -p 3306:3306 -d mysql:8.0
```

### 2. Set Environment Variables
```bash
export MYSQL_HOST=localhost
export MYSQL_USER=root  
export MYSQL_PASSWORD=demo123
export MYSQL_DATABASE=ecommerce
```

### 3. Database Schema
Our demo uses an e-commerce database with realistic sample data:
- **users**: Customer information (8 users from different countries)
- **categories**: Product categories with hierarchical structure
- **products**: 10 products with stock, pricing, and status
- **orders**: Customer orders with various statuses  
- **order_items**: Individual items within orders

---

## 🚀 Demo Scenarios
