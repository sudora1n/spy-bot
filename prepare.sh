set -e

cp -n .env.example .env
cp -n prometheus.yml.example prometheus.yml
cp -n compose-dev.yml compose.yml

bash ./scripts/generate_ed25519.sh
