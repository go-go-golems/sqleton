-- Create sample ecommerce database
USE ecommerce;

-- Create users table
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    status ENUM('active', 'inactive', 'suspended') DEFAULT 'active'
);

-- Create products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50) NOT NULL,
    stock_quantity INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('active', 'discontinued', 'out_of_stock') DEFAULT 'active'
);

-- Create orders table
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10,2) NOT NULL,
    status ENUM('pending', 'confirmed', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    shipping_address TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Create order_items table
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10,2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Insert sample users
INSERT INTO users (username, email, first_name, last_name, last_login, status) VALUES
('johndoe', 'john@example.com', 'John', 'Doe', '2024-01-15 10:30:00', 'active'),
('janesmit', 'jane@example.com', 'Jane', 'Smith', '2024-01-14 14:22:00', 'active'),
('bobwils', 'bob@example.com', 'Bob', 'Wilson', '2024-01-10 09:15:00', 'active'),
('alicebrown', 'alice@example.com', 'Alice', 'Brown', '2024-01-12 16:45:00', 'active'),
('charli_j', 'charlie@example.com', 'Charlie', 'Johnson', '2023-12-20 11:30:00', 'inactive'),
('susandav', 'susan@example.com', 'Susan', 'Davis', '2024-01-13 13:20:00', 'active'),
('mikemiller', 'mike@example.com', 'Mike', 'Miller', '2024-01-11 08:40:00', 'suspended'),
('emilywils', 'emily@example.com', 'Emily', 'Wilson', '2024-01-16 12:10:00', 'active');

-- Insert sample products
INSERT INTO products (name, description, price, category, stock_quantity, status) VALUES
('Wireless Headphones', 'High-quality Bluetooth headphones with noise cancellation', 129.99, 'Electronics', 45, 'active'),
('Coffee Maker', 'Programmable drip coffee maker with 12-cup capacity', 89.99, 'Appliances', 23, 'active'),
('Running Shoes', 'Lightweight running shoes with advanced cushioning', 149.99, 'Sports', 67, 'active'),
('Smartphone Case', 'Protective case for latest smartphone models', 24.99, 'Electronics', 120, 'active'),
('Yoga Mat', 'Premium non-slip yoga mat for all fitness levels', 39.99, 'Sports', 89, 'active'),
('Desk Lamp', 'LED desk lamp with adjustable brightness and USB charging', 79.99, 'Office', 34, 'active'),
('Backpack', 'Waterproof hiking backpack with multiple compartments', 199.99, 'Outdoor', 12, 'active'),
('Bluetooth Speaker', 'Portable wireless speaker with 360-degree sound', 69.99, 'Electronics', 56, 'active'),
('Kitchen Knife Set', 'Professional 8-piece knife set with cutting board', 159.99, 'Kitchen', 18, 'active'),
('Gaming Mouse', 'High-precision gaming mouse with RGB lighting', 89.99, 'Electronics', 0, 'out_of_stock'),
('Vintage Camera', 'Collectible vintage film camera from the 1970s', 299.99, 'Photography', 3, 'discontinued'),
('Organic Tea Set', 'Premium organic tea collection with 12 flavors', 49.99, 'Food', 78, 'active');

-- Insert sample orders
INSERT INTO orders (user_id, order_date, total_amount, status, shipping_address) VALUES
(1, '2024-01-10 14:30:00', 259.98, 'delivered', '123 Main St, Anytown, USA 12345'),
(2, '2024-01-11 09:15:00', 89.99, 'shipped', '456 Oak Ave, Somewhere, USA 67890'),
(3, '2024-01-12 16:20:00', 179.98, 'confirmed', '789 Pine Rd, Elsewhere, USA 54321'),
(1, '2024-01-13 11:45:00', 129.99, 'pending', '123 Main St, Anytown, USA 12345'),
(4, '2024-01-14 13:10:00', 109.98, 'delivered', '321 Elm St, Newtown, USA 98765'),
(6, '2024-01-15 10:30:00', 199.99, 'shipped', '654 Maple Dr, Oldtown, USA 13579'),
(8, '2024-01-16 15:45:00', 319.97, 'confirmed', '987 Cedar Ln, Midtown, USA 24680'),
(2, '2024-01-17 08:20:00', 24.99, 'pending', '456 Oak Ave, Somewhere, USA 67890');

-- Insert sample order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
-- Order 1: Wireless Headphones + Coffee Maker
(1, 1, 1, 129.99),
(1, 2, 1, 89.99),
(1, 4, 1, 24.99),
-- Order 2: Coffee Maker
(2, 2, 1, 89.99),
-- Order 3: Running Shoes + Yoga Mat
(3, 3, 1, 149.99),
(3, 5, 1, 39.99),
-- Order 4: Wireless Headphones
(4, 1, 1, 129.99),
-- Order 5: Desk Lamp + Yoga Mat
(5, 6, 1, 79.99),
(5, 5, 1, 39.99),
-- Order 6: Backpack
(6, 7, 1, 199.99),
-- Order 7: Bluetooth Speaker + Kitchen Knife Set + Organic Tea Set
(7, 8, 1, 69.99),
(7, 9, 1, 159.99),
(7, 12, 2, 49.99),
-- Order 8: Smartphone Case
(8, 4, 1, 24.99);
