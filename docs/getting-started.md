# Getting Started with Property Bookings API

Welcome to the Property Bookings API API! This guide will help you get up and running quickly.

## Overview

**API Version:** 1.0.0  
**Base URL:** `https://api.bookings-sample.dev/v1`

API to manage property bookings — create, update, cancel and query bookings. Each booking includes check-in/out dates, guest count, and price; designed for integration and AI readiness.

## Prerequisites

Before you begin, make sure you have:

- An API key or authentication credentials
- Access to the API environment
- Basic understanding of REST APIs

## Authentication

Most API endpoints require authentication. Include your API key in the request header:

```bash
Authorization: Bearer YOUR_API_KEY
```

## Quick Start

### 1. Get Your API Key

Contact your administrator to obtain your API key.

### 2. Make Your First Request

Here's a simple example to get you started:

```bash
curl -X GET \
     -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.bookings-sample.dev/v1/health
```

### 3. Explore the API

- Check out the API Reference for detailed endpoint documentation
- Review examples for code samples in different languages

## Rate Limits

Please be mindful of rate limits when making requests. Contact support if you need higher limits.

## Support

Need help? Contact support at support@bookings-sample.dev

## Next Steps

- Read the API Reference
- Try the Examples
- Set up Integration
