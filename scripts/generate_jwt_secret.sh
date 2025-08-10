#!/bin/bash

JWT_SECRET=$(openssl rand -base64 32)

# Output the result
echo "Generated JWT Secret Key:"
echo "$JWT_SECRET"
