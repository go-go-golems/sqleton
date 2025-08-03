-- E-commerce Demo Database Setup for sqleton README examples

USE ecommerce;

-- Drop tables if they exist (for clean setup)
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;

-- Create users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    phone VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status ENUM('active', 'inactive', 'suspended') DEFAULT 'active',
    country VARCHAR(50) DEFAULT 'US'
);

-- Create categories table
CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES categories(id)
);

-- Create products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock_quantity INT DEFAULT 0,
    category_id INT,
    sku VARCHAR(50) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status ENUM('active', 'inactive', 'discontinued') DEFAULT 'active',
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- Create orders table
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    total_amount DECIMAL(10,2) NOT NULL,
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    shipping_address TEXT,
    payment_method VARCHAR(50),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create order_items table
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Insert sample users
INSERT INTO users (username, email, first_name, last_name, phone, status, country, created_at) VALUES
('jsmith', 'john.smith@email.com', 'John', 'Smith', '+1-555-0101', 'active', 'US', '2024-01-15 10:30:00'),
('mjohnson', 'mary.johnson@email.com', 'Mary', 'Johnson', '+1-555-0102', 'active', 'US', '2024-01-20 14:20:00'),
('bwilson', 'bob.wilson@email.com', 'Bob', 'Wilson', '+1-555-0103', 'active', 'CA', '2024-02-01 09:15:00'),
('achang', 'alice.chang@email.com', 'Alice', 'Chang', '+1-555-0104', 'active', 'US', '2024-02-10 16:45:00'),
('dgarcia', 'david.garcia@email.com', 'David', 'Garcia', '+1-555-0105', 'inactive', 'MX', '2024-02-15 11:30:00'),
('sbrown', 'sarah.brown@email.com', 'Sarah', 'Brown', '+44-207-0106', 'active', 'UK', '2024-03-01 13:20:00'),
('mlee', 'mike.lee@email.com', 'Mike', 'Lee', '+1-555-0107', 'suspended', 'US', '2024-03-05 08:10:00'),
('ewhite', 'emma.white@email.com', 'Emma', 'White', '+33-1-0108', 'active', 'FR', '2024-03-10 15:30:00');

-- Insert categories
INSERT INTO categories (name, description, parent_id) VALUES
('Electronics', 'Electronic devices and accessories', NULL),
('Computers', 'Laptops, desktops, and computer accessories', 1),
('Mobile Phones', 'Smartphones and mobile accessories', 1),
('Clothing', 'Fashion and apparel', NULL),
('Men\'s Clothing', 'Clothing for men', 4),
('Women\'s Clothing', 'Clothing for women', 4),
('Books', 'Physical and digital books', NULL),
('Home & Garden', 'Home improvement and garden supplies', NULL);

-- Insert products
INSERT INTO products (name, description, price, stock_quantity, category_id, sku, status, created_at) VALUES
('MacBook Pro 14"', 'Apple MacBook Pro with M2 chip, 14-inch display', 1999.99, 25, 2, 'MBP-14-M2', 'active', '2024-01-10 12:00:00'),
('iPhone 15 Pro', 'Latest iPhone with advanced camera system', 999.99, 50, 3, 'IP15-PRO', 'active', '2024-01-15 10:00:00'),
('Samsung Galaxy S24', 'Android smartphone with AI features', 799.99, 30, 3, 'SGS24', 'active', '2024-01-20 14:00:00'),
('Dell XPS 13', 'Ultra-thin laptop with Intel Core i7', 1299.99, 15, 2, 'XPS13-I7', 'active', '2024-02-01 09:00:00'),
('Men\'s T-Shirt', 'Cotton t-shirt in various colors', 19.99, 100, 5, 'MTS-COT', 'active', '2024-02-05 11:00:00'),
('Women\'s Dress', 'Elegant evening dress', 89.99, 25, 6, 'WD-EVEN', 'active', '2024-02-10 13:00:00'),
('Programming Book', 'Learn Go Programming', 39.99, 75, 7, 'BK-GO-PROG', 'active', '2024-02-15 15:00:00'),
('Garden Tools Set', 'Complete gardening tool kit', 149.99, 20, 8, 'GTS-COMP', 'active', '2024-03-01 10:00:00'),
('Vintage Camera', 'Classic film camera (discontinued)', 299.99, 0, 1, 'CAM-VINT', 'discontinued', '2023-12-01 12:00:00'),
('Wireless Headphones', 'Noise-cancelling Bluetooth headphones', 199.99, 40, 1, 'WH-BT-NC', 'active', '2024-03-05 14:00:00');

-- Insert orders
INSERT INTO orders (user_id, total_amount, status, shipping_address, payment_method, created_at) VALUES
(1, 2019.98, 'delivered', '123 Main St, Anytown, US 12345', 'credit_card', '2024-01-25 10:30:00'),
(2, 999.99, 'shipped', '456 Oak Ave, Springfield, US 67890', 'paypal', '2024-02-05 14:20:00'),
(3, 1499.98, 'processing', '789 Pine Rd, Vancouver, CA V6B 1A1', 'credit_card', '2024-02-15 09:15:00'),
(4, 59.98, 'delivered', '321 Elm St, Portland, US 97201', 'debit_card', '2024-02-20 16:45:00'),
(6, 349.98, 'pending', '654 Maple Dr, London, UK SW1A 1AA', 'bank_transfer', '2024-03-01 11:30:00'),
(1, 149.99, 'shipped', '123 Main St, Anytown, US 12345', 'credit_card', '2024-03-05 13:20:00'),
(8, 199.99, 'delivered', '987 Cedar Ln, Paris, FR 75001', 'credit_card', '2024-03-08 15:30:00'),
(2, 39.99, 'cancelled', '456 Oak Ave, Springfield, US 67890', 'paypal', '2024-03-10 08:10:00');

-- Insert order items
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
-- Order 1: MacBook + T-shirt
(1, 1, 1, 1999.99),
(1, 5, 1, 19.99),
-- Order 2: iPhone
(2, 2, 1, 999.99),
-- Order 3: Dell laptop + Headphones
(3, 4, 1, 1299.99),
(3, 10, 1, 199.99),
-- Order 4: T-shirt + Book
(4, 5, 2, 19.99),
(4, 7, 1, 39.99),
-- Order 5: Galaxy + Dress
(5, 3, 1, 799.99),
(5, 6, 1, 89.99),
-- Order 6: Garden tools
(6, 8, 1, 149.99),
-- Order 7: Headphones
(7, 10, 1, 199.99),
-- Order 8: Book (cancelled)
(8, 7, 1, 39.99);
