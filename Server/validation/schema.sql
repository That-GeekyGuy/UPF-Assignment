-- Create the database if it doesn't exist
CREATE DATABASE IF NOT EXISTS upf;

USE upf;

-- IMSI table to store IMSI information
CREATE TABLE IF NOT EXISTS imsi (
    id INT AUTO_INCREMENT PRIMARY KEY,
    imsi_number VARCHAR(15) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- FSEID (Fully Qualified Session Endpoint ID) table
CREATE TABLE IF NOT EXISTS fseid (
    id INT AUTO_INCREMENT PRIMARY KEY,
    imsi_id INT NOT NULL,
    fseid_value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (imsi_id) REFERENCES imsi(id) ON DELETE CASCADE,
    UNIQUE KEY unique_fseid (fseid_value, imsi_id)
);

-- PDR (Packet Detection Rule) table
CREATE TABLE IF NOT EXISTS pdr (
    id INT AUTO_INCREMENT PRIMARY KEY,
    fseid_id INT NOT NULL,
    pdr_id VARCHAR(100) NOT NULL,
    dnn VARCHAR(50) NOT NULL,
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (fseid_id) REFERENCES fseid(id) ON DELETE CASCADE,
    UNIQUE KEY unique_pdr (fseid_id, pdr_id)
);

-- Add index for better query performance
CREATE INDEX idx_imsi_number ON imsi(imsi_number);
CREATE INDEX idx_pdr_status ON pdr(status);
