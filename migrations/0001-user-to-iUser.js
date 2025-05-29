const DEFAULT_BOT_ID = NumberLong(1);

print("1. creating indexes...");
db.bot_users.createIndex({ user_id: 1, bot_id: 1 }, { unique: true });

function getNextBotUserSeq() {
  const res = db.counters.findOneAndUpdate(
    { _id: "bot_users" },
    { $inc: { value: NumberLong(1) } },
    { returnDocument: "after", upsert: true },
  );
  return res.value;
}

print("2. users → bot_users...");
const cursor = db.users.find();
let errors = [];

cursor.forEach((oldDoc) => {
  try {
    const botCreatedAt = oldDoc.created_at || Date.now();

    const newBotUser = {
      _id: getNextBotUserSeq(),
      user_id: oldDoc._id,
      bot_id: DEFAULT_BOT_ID,
      business_connections: oldDoc.business_connections || [],
      send_messages:
        oldDoc.send_messages === undefined ? true : oldDoc.send_messages,
      created_at: botCreatedAt,
    };

    db.bot_users.insertOne(newBotUser);

    db.users.updateOne(
      { _id: oldDoc._id },
      { $unset: { business_connections: "", send_messages: "" } },
    );

    print(`done: user ${oldDoc._id} -> botUser ${newBotUser._id}`);
  } catch (e) {
    print(`error while migration user ${oldDoc._id}: ${e}`);
    errors.push({ user: oldDoc._id, error: e.toString() });
  }
});

if (errors.length > 0) {
  print("\n---migration done with errors:---");
  printjson(errors);
} else {
  print("\n--- migration done without errors ---");
}

print("\n3. results of migration:");
print(`len of users: ${db.users.countDocuments()}`);
print(`len of bot_users: ${db.bot_users.countDocuments()}`);
print(`counter bot_users: ${db.counters.findOne({ _id: "bot_users" }).value}`);

print("\n4. copying send_messages → main_send_messages...");

const botCursor = db.bot_users.find({ bot_id: DEFAULT_BOT_ID });
let updated = 0;
let copyErrors = [];

botCursor.forEach((botUser) => {
  try {
    const res = db.users.updateOne(
      { _id: botUser.user_id },
      { $set: { creator_send_messages: botUser.send_messages } },
    );

    if (res.matchedCount > 0) {
      updated += 1;
    } else {
      throw `user not found: ${botUser.user_id}`;
    }
  } catch (e) {
    print(`error while update user: ${botUser.user_id}: ${e}`);
    copyErrors.push({ user: botUser.user_id, error: e.toString() });
  }
});

print(`\n done. updated: ${updated}`);
if (copyErrors.length > 0) {
  print("--- errors while copy: ---");
  printjson(copyErrors);
} else {
  print("--- done without errors ---");
}
