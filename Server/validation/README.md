# Validation Server

The Validation Server is a RESTful API service that handles validation and management of PDR (Packet Detection Rules) for IMSIs (International Mobile Subscriber Identities) in a UPF (User Plane Function) environment.

## API Endpoints

### Health Check
- `GET /health` - Check if the server is running

### Validation Endpoints
- `GET /validate?imsi=<imsi>` - Get all PDRs for an IMSI
- `POST /validate` - Validate a PDR for an IMSI
- `PUT /validate` - Update a PDR for an IMSI
- `DELETE /validate?imsi=<imsi>&pdr_id=<pdr_id>` - Delete a PDR for an IMSI

## Request/Response Examples

### GET /validate
**Request:**
```
GET /validate?imsi=001011234567890
```

**Response:**
```json
{
  "status": "success",
  "imsi": "001011234567890",
  "internet_pdrs": ["pdr1", "pdr2"],
  "ims_pdrs": ["pdr3"],
  "timestamp": "2025-07-17T11:30:45Z"
}
```

### POST /validate
**Request:**
```json
{
  "imsi": "001011234567890",
  "rules": {
    "pdr_id": "pdr4",
    "dnn": "internet"
  }
}
```

**Response (Success):**
```json
{
  "status": "success",
  "message": "PDR found",
  "imsi": "001011234567890",
  "pdr": "pdr4",
  "dnn": "internet",
  "found_in": "internet",
  "timestamp": "2025-07-17T11:31:22Z"
}
```

## Database Schema

Run the `schema.sql` script to set up the required database tables:
```bash
mysql -u username -p < validation/schema.sql
```

## Starting the Server

The Validation Server is automatically started as part of the main Server application. It runs on port 8080 by default.

## Dependencies

- Go 1.16+
- MySQL 5.7+
- Gin Web Framework
- Go MySQL Driver
