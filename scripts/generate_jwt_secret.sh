#!/bin/bash

JWT_SECRET=$(openssl rand -base64 32)

echo "Generated JWT Secret Key:"
echo "$JWT_SECRET"
