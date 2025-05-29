# How to migrations

docker run -it --rm  --network=ssuspy_ssuspy -v $(pwd)/migrations:/scripts  mongodb/mongodb-community-server:latest mongosh "MONGODB_URL" /scripts/some.js
