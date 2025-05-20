db = db.getSiblingDB('ssuspy');
db.createUser({
  user: process.env.MONGO_USERNAME,
  pwd:  process.env.MONGO_PASSWORD,
  roles: [{ role: 'readWrite', db: 'ssuspy' }]
});